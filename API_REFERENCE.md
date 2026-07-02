# Iru (Kandji) API reference — clean-room notes

Facts extracted from the Iru Endpoint Management API Postman collection and verified
against the live tenant (read-only). These are API facts (endpoints, JSON shapes,
enums) used to build the provider independently. Not derived from any other provider's
source.

Base URL: `https://<subdomain>.api.kandji.io` (US) / `.api.eu.kandji.io` (EU).
Auth: `Authorization: Bearer <token>`. List envelope: `{count, next, previous, results[]}`
with `?page=` / `?limit=&offset=` pagination (limit max 300).

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
1. POST `/upload` (JSON `{filename: "<file>"}`) → presigned S3
2. POST to S3 → 204
3. Poll GET `/upload/{pending_upload_id}/status` until `VALIDATED`
   (terminal failures: `UPLOAD_FAILED`, `VALIDATE_FAILED`)
4. POST `/library/ipa-apps` (JSON): `file_key, name, runs_on_iphone, runs_on_ipad,
   runs_on_tv, active`
- PATCH `/{id}` (JSON); pass `file_key` only to replace.

## Blueprints — /api/v1/blueprints
- POST (urlencoded): `name, color, description, icon, type (map),
  enrollment_code.code, enrollment_code.is_active, source.type, source.id`
- PATCH `/{id}` (urlencoded). GET list / GET `/{id}` / DELETE `/{id}`.
- GET `/{id}/list-library-items` — items assigned to a blueprint.
- POST `/{id}/assign-library-item` (JSON): `{library_item_id, assignment_node_id?}`
  (`assignment_node_id` not required for maps without conditional logic).
