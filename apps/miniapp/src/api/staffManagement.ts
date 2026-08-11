import type {
  IdentityAdminApi,
  OrganizationAdminApi,
  OrganizationDirectoryApi,
  StaffManagementApi,
} from "@/api/management/contracts";
import { httpIdentityAdminApi } from "@/api/management/identityAdmin.http";
import { httpOrganizationAdminApi } from "@/api/management/organizationAdmin.http";
import { httpOrganizationDirectoryApi } from "@/api/management/organizationDirectory.http";

import { httpStaffManagementApi } from "./staffManagement.http";

export type {
  IdentityAdminApi,
  OrganizationAdminApi,
  OrganizationDirectoryApi,
  StaffManagementApi,
} from "@/api/management/contracts";

export const organizationDirectoryApi: OrganizationDirectoryApi =
  httpOrganizationDirectoryApi;
export const organizationAdminApi: OrganizationAdminApi = httpOrganizationAdminApi;
export const identityAdminApi: IdentityAdminApi = httpIdentityAdminApi;

// Kept for compatibility while pages migrate to domain-focused APIs.
export const staffManagementApi: StaffManagementApi = httpStaffManagementApi;
