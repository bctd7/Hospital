-- 暂时移除引用旧科室表的外键，完成表结构演进后重新建立。
ALTER TABLE identity_staff_profiles
    DROP FOREIGN KEY fk_identity_staff_department;

-- 移除旧表自身的父子关系外键，避免重命名和索引调整互相影响。
ALTER TABLE identity_departments
    DROP FOREIGN KEY fk_identity_departments_parent;

-- 科室表演进为统一组织单元表。
RENAME TABLE identity_departments TO identity_organization_units;

ALTER TABLE identity_organization_units
    ADD COLUMN unit_type VARCHAR(16) NOT NULL AFTER parent_id,
    ADD COLUMN version BIGINT UNSIGNED NOT NULL DEFAULT 1 AFTER status,

    DROP INDEX uk_identity_departments_code,
    DROP INDEX idx_identity_departments_parent,
    DROP CHECK chk_identity_departments_status,

    ADD UNIQUE KEY uk_identity_organization_units_parent_code (parent_id, code),
    ADD KEY idx_identity_organization_units_parent_type_status
        (parent_id, unit_type, status, code, id),

    ADD CONSTRAINT chk_identity_organization_units_type
        CHECK (unit_type IN ('hospital', 'campus', 'department')),
    ADD CONSTRAINT chk_identity_organization_units_status
        CHECK (status IN ('active', 'disabled')),
    ADD CONSTRAINT chk_identity_organization_units_version
        CHECK (version > 0),

    ADD CONSTRAINT fk_identity_organization_units_parent
        FOREIGN KEY (parent_id)
            REFERENCES identity_organization_units (id);

-- 工作人员当前科室改为引用统一组织单元表。
-- “目标必须是 department”无法通过普通外键表达，后续由业务层校验。
ALTER TABLE identity_staff_profiles
    ADD CONSTRAINT fk_identity_staff_organization_unit
        FOREIGN KEY (department_id)
            REFERENCES identity_organization_units (id);
