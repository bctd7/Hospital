-- 先移除引用统一组织单元表的工作人员外键。
ALTER TABLE identity_staff_profiles
DROP FOREIGN KEY fk_identity_staff_organization_unit;

-- 移除组织单元自身的父子关系外键。
ALTER TABLE identity_organization_units
DROP FOREIGN KEY fk_identity_organization_units_parent;

-- 移除 000003 增加的索引和约束。
ALTER TABLE identity_organization_units
DROP INDEX uk_identity_organization_units_parent_code,
DROP INDEX idx_identity_organization_units_parent_type_status,
DROP CHECK chk_identity_organization_units_type,
    DROP CHECK chk_identity_organization_units_status,
    DROP CHECK chk_identity_organization_units_version;

-- 恢复旧科室表拥有的字段、索引和状态约束。
ALTER TABLE identity_organization_units
DROP COLUMN version,
    DROP COLUMN unit_type,

    ADD UNIQUE KEY uk_identity_departments_code (code),
    ADD KEY idx_identity_departments_parent (parent_id),

    ADD CONSTRAINT chk_identity_departments_status
        CHECK (status IN ('active', 'disabled'));

RENAME TABLE identity_organization_units TO identity_departments;

-- 恢复旧科室表父子关系。
ALTER TABLE identity_departments
    ADD CONSTRAINT fk_identity_departments_parent
        FOREIGN KEY (parent_id)
            REFERENCES identity_departments (id);

-- 恢复工作人员到旧科室表的外键。
ALTER TABLE identity_staff_profiles
    ADD CONSTRAINT fk_identity_staff_department
        FOREIGN KEY (department_id)
            REFERENCES identity_departments (id);
