import type {
  IdentityAdminApi,
  OrganizationAdminApi,
  OrganizationDirectoryApi,
} from "@/api/management/contracts";
import { httpIdentityAdminApi } from "@/api/management/identityAdmin.http";
import { httpOrganizationAdminApi } from "@/api/management/organizationAdmin.http";
import { httpOrganizationDirectoryApi } from "@/api/management/organizationDirectory.http";

export type {
  IdentityAdminApi,
  OrganizationAdminApi,
  OrganizationDirectoryApi,
} from "@/api/management/contracts";

export const organizationDirectoryApi: OrganizationDirectoryApi =
  httpOrganizationDirectoryApi;
export const organizationAdminApi: OrganizationAdminApi = httpOrganizationAdminApi;
export const identityAdminApi: IdentityAdminApi = httpIdentityAdminApi;
