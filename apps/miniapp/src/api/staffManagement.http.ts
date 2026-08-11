import type { StaffManagementApi } from "@/api/management/contracts";
import { httpIdentityAdminApi } from "@/api/management/identityAdmin.http";
import { httpOrganizationAdminApi } from "@/api/management/organizationAdmin.http";
import { httpOrganizationDirectoryApi } from "@/api/management/organizationDirectory.http";

// Compatibility facade for existing callers. Domain-focused code should use
// the narrower APIs exported from staffManagement.ts.
export const httpStaffManagementApi: StaffManagementApi = {
  ...httpOrganizationAdminApi,
  ...httpIdentityAdminApi,
  getOrganizationContext: () => httpOrganizationDirectoryApi.getOrganizationContext(),
  listDepartments: (campusId, includeDisabled = false) =>
    includeDisabled
      ? httpOrganizationAdminApi.listDepartments(campusId, true)
      : httpOrganizationDirectoryApi.listDepartments(campusId),
  listDoctors: (departmentId) => httpOrganizationDirectoryApi.listDoctors(departmentId),
};
