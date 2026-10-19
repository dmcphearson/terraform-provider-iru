# Iru (Kandji) API reference — clean-room notes

Facts extracted from the Iru Endpoint Management API Postman collection and verified
against the live tenant (read-only). These are API facts (endpoints, JSON shapes,
enums) used to build the provider independently. Not derived from any other provider's
source.

Base URL: `https://<subdomain>.api.kandji.io` (US) / `.api.eu.kandji.io` (EU).
Auth: `Authorization: Bearer <token>`. List envelope: `{count, next, previous, results[]}`
with `?page=` / `?limit=&offset=` pagination (limit max 300).

## Self Service rules (verified live, scripts and custom apps)
- `no_enforcement` requires `show_in_self_service=true`
  (400 `{"show_in_self_service":["Required when not enforcing install"]}`).
- `self_service_recommended` is rejected, even as false, unless Self Service is on
  (400 `{"self_service_recommended":["Cannot set without enabling self service"]}`).
- Assigning an inactive item to a blueprint fails (400 `"Library Item is not active."`).

## Custom Scripts — /api/v1/library/custom-scripts
- POST (JSON) / PATCH `/{id}` (JSON) / GET list / GET `/{id}` / DELETE `/{id}` (204)
- Write fields: `name`, `execution_frequency` (enum: `once|every_15_min|every_day|no_enforcement`),
  `script`, `remediation_script?`, `show_in_self_service?`, `self_service_category_id?`,
  `self_service_recommended?`, `active?`, `restart?`
- GET returns: `id, name, active, execution_frequency, restart, script,
  remediation_script (""), created_at, updated_at, show_in_self_service`
- **WRITE-ONLY (verified live):** `self_service_category_id` and `self_service_recommended`
  are accepted on write but NOT returned by GET → model as plain Optional, carry prior
  state value forward on Read (else perpetual drift).
- **Trailing whitespace:** API trims trailing newline/space from `script`. Preserve the
  configured value in state; do not overwrite from the (trimmed) response.

## Custom Profiles — /api/v1/library/custom-profiles
- POST/PATCH multipart formdata: `name` (text), `file` (file, the .mobileconfig),
  `active`, `runs_on_mac`, `runs_on_iphone`, `runs_on_ipad`, `runs_on_tv` (text bools)
- GET returns: `id, name, active, mdm_identifier, profile (raw XML string),
  runs_on_mac/iphone/ipad/tv/vision, created_at, updated_at`
- **`profile` (XML) is returned but the input `file` is NOT echoed as a re-uploadable
  field** → `profile_file` (our input attr) is not read back; expect one-time benign
  diff on import. `runs_on_vision` exists on this tenant (add as optional).

## Tags — /api/v1/tags
- POST (JSON `{name}`) / PATCH `/{id}` (JSON `{name}`) / GET list (`?search=`) / DELETE `/{id}`
- **NO GET-by-id endpoint.** Read = list + filter by id; RemoveResource if absent.
- GET list item: `{id, name}`

## Self Service Categories — /api/v1/self-service/categories
- GET only. **Returns a flat JSON array** `[{id, name}, ...]` (NOT a results envelope).
- Read-only; supplies `self_service_category_id` values.

## Custom Apps — /api/v1/library/custom-apps  (3-step upload)
1. POST `/upload` (JSON `{name: "<file>"}`) → `{name, expires, post_url, post_data{...}, file_key}`
2. POST `post_url` (multipart to S3: echo every `post_data` field + `file`) → 204
3. POST `/library/custom-apps` (**urlencoded**): `file_key, name, install_type,
   install_enforcement, unzip_location?, audit_script?, preinstall_script?,
   postinstall_script?, show_in_self_service?, self_service_category_id?,
   self_service_recommended?, active?, restart?`
- PATCH `/{id}` (urlencoded); pass `file_key` only when replacing the binary.
- GET returns: `id, name, file_key, install_type, install_enforcement, audit_script,
  unzip_location, active, restart, preinstall_script, postinstall_script, file_url,
  file_size, file_updated, sha256, created_at, updated_at, show_in_self_service,
  self_service_category_id, self_service_recommended`
- **self_service fields ARE returned here (unlike scripts)** → Optional+Computed round-trip.
- **`sha256` is returned** → use for binary change detection (re-upload only on change).

## IPA Apps — /api/v1/library/ipa-apps  (4-step upload)
1. POST `/upload` (JSON `{filename: "<file>"}`) → `{id, filename, sha256, expires,
   post_url, post_data{...}, file_key}`; `id` is the pending upload id
2. POST to S3 → 204
3. Poll GET `/upload/{pending_upload_id}/status` (~1s) until `VALIDATED`; `UPLOADING`
   and `VALIDATING` are in progress, `UPLOAD_FAILED` / `VALIDATE_FAILED` are terminal
4. POST `/library/ipa-apps` (JSON): `file_key, name, runs_on_iphone, runs_on_ipad,
   runs_on_tv, active` → 201, no documented body
- PATCH `/{id}` (JSON); pass `file_key` only to replace. No documented body.
- GET returns: `id, name, file_key, active, icon, ipad_icon, identifier, app_identifier,
  app_name, app_version, supported_device_families[], minimum_os_version, file_url,
  sha256, file_size, file_updated, created_at, updated_at, runs_on_iphone, runs_on_ipad,
  runs_on_tv, runs_on_mac, runs_on_ipod, runs_on_watch`

## Blueprints — /api/v1/blueprints
- POST (urlencoded): `name, color, description, icon, type (map only; "classic" →
  400 "Invalid blueprint type", verified live),
  enrollment_code.code, enrollment_code.is_active, source.type, source.id`
- PATCH `/{id}` (urlencoded). GET list / GET `/{id}` / DELETE `/{id}`.
- GET `/{id}/list-library-items` — items assigned to a blueprint.
- POST `/{id}/assign-library-item` (JSON): `{library_item_id, assignment_node_id?}`
  (`assignment_node_id` not required for maps without conditional logic).

## Devices — /api/v1/devices  (reads and notes verified live; device PATCH not)
- GET list returns a **bare array** (no envelope); `limit` (max 300) / `offset` paging.
  Filters: `blueprint_id`, `platform` (`Mac|iPad|iPhone|AppleTV`), `serial_number`,
  `device_name`, `user_email`, `user_name` (substring), `asset_tag`, `model`,
  `os_version`, `mac_address`, `filevault_enabled`, `ordering`.
- GET `/{device_id}`: `device_id, device_name, model, serial_number, platform, os_version,
  last_check_in, user{id,email,name,is_archived}, asset_tag, blueprint_id, blueprint_name,
  mdm_enabled, agent_installed, agent_version, is_missing, is_removed, first_enrollment,
  last_enrollment, tags[]`. Polymorphic: non-applicable fields are blank. `user` is `""`
  when unassigned.
- GET `/{device_id}/details`: large nested record, shape varies by platform.
- PATCH `/{device_id}`, secrets under `/{device_id}/secrets/*`.
- Notes: GET `/{device_id}/notes` (bare array), POST `{content}` (201), GET, PATCH `{content}`
  and DELETE `/{device_id}/notes/{note_id}`. Item: `note_id, created_at, updated_at, author,
  content`. Notes written in the UI come back as HTML (`<p>...&nbsp;</p>`).

## Blueprint Routing — /api/v1/blueprint-routing/  (GET and activity verified live; PATCH not)
- One setting per tenant. Every endpoint returns 404 while routing is disabled on the tenant.
  GET/PATCH `/`: `{enrollment_code{code,is_active}}`; PATCH takes either key.
  Routing rules are not exposed.
- GET `/activity`: `{count,next,previous,results[]}`, limit/offset (max 300). Item: `id` (int),
  `activity_time`, `activity_type` (`routing_departure`, `computer_enrollment`, `enrollment`,
  `routing_config_edited`, ...), `user{id,email}` or null, `device_id` or null, `details` (shape
  varies by type).

## Users — /api/v1/users  (verified live)
- GET list: `{next, previous, results[]}` with a **cursor** in `next` (absolute URL; may
  name the web host, not the API host). Default 25 per page. Filters: `email`, `id`,
  `integration_id`, `archived`.
- Item: `id, name, email, active, archived, department, job_title, device_count,
  deprecated_user_id, created_at, updated_at, integration{id,uuid,name,type}` (nullable).
- GET `/{user_id}`, DELETE `/{user_id}`.
