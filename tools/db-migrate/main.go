package main

import (
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	gomysql "github.com/go-sql-driver/mysql"
	"github.com/golang-migrate/migrate/v4"
	migratemysql "github.com/golang-migrate/migrate/v4/database/mysql"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

const databaseDriverName = "mysql"

type options struct {
	migrationsPath string
	dsnEnvironment string
	direction      string
	steps          int
	baseline       uint
}

func main() {
	if err := run(parseOptions()); err != nil {
		fmt.Fprintln(os.Stderr, "migration failed:", err)
		os.Exit(1)
	}
}

func parseOptions() options {
	var opts options
	flag.StringVar(&opts.migrationsPath, "migrations", "", "directory containing migration SQL files")
	flag.StringVar(&opts.dsnEnvironment, "dsn-env", "", "environment variable containing the MySQL DSN")
	flag.StringVar(&opts.direction, "direction", "up", "migration operation: up, down, version, or baseline")
	flag.IntVar(&opts.steps, "steps", 1, "number of migrations to roll back")
	flag.UintVar(&opts.baseline, "baseline-version", 0, "existing schema version to record during baseline")
	flag.Parse()
	return opts
}

func run(opts options) error {
	if opts.migrationsPath == "" {
		return errors.New("-migrations is required")
	}
	if opts.dsnEnvironment == "" {
		return errors.New("-dsn-env is required")
	}

	dsn := strings.TrimSpace(os.Getenv(opts.dsnEnvironment))
	if dsn == "" {
		return fmt.Errorf("environment variable %s is empty", opts.dsnEnvironment)
	}

	migrationURL, err := migrationSourceURL(opts.migrationsPath)
	if err != nil {
		return err
	}

	database, err := openDatabase(dsn)
	if err != nil {
		return err
	}
	defer database.Close()

	driver, err := migratemysql.WithInstance(database, &migratemysql.Config{})
	if err != nil {
		return fmt.Errorf("initialize MySQL migration driver: %w", err)
	}

	runner, err := migrate.NewWithDatabaseInstance(migrationURL, databaseDriverName, driver)
	if err != nil {
		return fmt.Errorf("initialize migration runner: %w", err)
	}
	defer closeRunner(runner)

	switch opts.direction {
	case "up":
		return runUp(runner)
	case "down":
		return runDown(runner, opts.steps)
	case "version":
		return printVersion(runner)
	case "baseline":
		return runBaseline(runner, opts.baseline)
	default:
		return fmt.Errorf("unsupported direction %q", opts.direction)
	}
}

func openDatabase(rawDSN string) (*sql.DB, error) {
	config, err := gomysql.ParseDSN(rawDSN)
	if err != nil {
		return nil, fmt.Errorf("parse MySQL DSN: %w", err)
	}
	config.MultiStatements = true

	database, err := sql.Open(databaseDriverName, config.FormatDSN())
	if err != nil {
		return nil, fmt.Errorf("open MySQL connection: %w", err)
	}
	if err := database.Ping(); err != nil {
		database.Close()
		return nil, fmt.Errorf("connect to MySQL: %w", err)
	}
	return database, nil
}

func migrationSourceURL(path string) (string, error) {
	absolutePath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("resolve migration path: %w", err)
	}
	info, err := os.Stat(absolutePath)
	if err != nil {
		return "", fmt.Errorf("inspect migration path: %w", err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("migration path %s is not a directory", absolutePath)
	}

	urlPath := filepath.ToSlash(absolutePath)
	volume := filepath.ToSlash(filepath.VolumeName(absolutePath))
	if volume != "" {
		urlPath = strings.TrimPrefix(urlPath, volume)
		return (&url.URL{Scheme: "file", Host: volume, Path: urlPath}).String(), nil
	}
	return (&url.URL{Scheme: "file", Path: urlPath}).String(), nil
}

func runUp(runner *migrate.Migrate) error {
	err := runner.Up()
	if errors.Is(err, migrate.ErrNoChange) {
		fmt.Println("database is already at the latest migration version")
		return nil
	}
	if err != nil {
		return fmt.Errorf("apply migrations: %w", err)
	}
	return printVersion(runner)
}

func runDown(runner *migrate.Migrate, steps int) error {
	if steps < 1 {
		return errors.New("-steps must be at least 1 for down migrations")
	}
	if err := runner.Steps(-steps); err != nil {
		return fmt.Errorf("roll back %d migration(s): %w", steps, err)
	}
	return printVersion(runner)
}

func runBaseline(runner *migrate.Migrate, version uint) error {
	if version == 0 {
		return errors.New("-baseline-version must be greater than zero")
	}

	currentVersion, dirty, err := runner.Version()
	if err == nil {
		return fmt.Errorf("database already has migration version %d (dirty=%t)", currentVersion, dirty)
	}
	if !errors.Is(err, migrate.ErrNilVersion) {
		return fmt.Errorf("read current migration version before baseline: %w", err)
	}
	if err := runner.Force(int(version)); err != nil {
		return fmt.Errorf("record baseline version %d: %w", version, err)
	}
	fmt.Printf("recorded existing schema as migration version %d; no schema SQL was executed\n", version)
	return nil
}

func printVersion(runner *migrate.Migrate) error {
	version, dirty, err := runner.Version()
	if errors.Is(err, migrate.ErrNilVersion) {
		fmt.Println("migration version: none")
		return nil
	}
	if err != nil {
		return fmt.Errorf("read migration version: %w", err)
	}
	fmt.Printf("migration version: %d (dirty=%t)\n", version, dirty)
	if dirty {
		return errors.New("database migration state is dirty; manual inspection is required")
	}
	return nil
}

func closeRunner(runner *migrate.Migrate) {
	sourceErr, databaseErr := runner.Close()
	if sourceErr != nil {
		fmt.Fprintln(os.Stderr, "warning: close migration source:", sourceErr)
	}
	if databaseErr != nil {
		fmt.Fprintln(os.Stderr, "warning: close migration database driver:", databaseErr)
	}
}
