def release_major:
  capture("^v(?<major>[0-9]+)\\.[0-9]+\\.[0-9]+$").major | tonumber;

type == "object" and
(keys == $expected_keys) and
all(to_entries[];
  (.value | type == "object") and
  (.value | keys == ["directory", "module", "module_version", "ref", "repository", "version"]) and
  (.value.directory | test("^sitectl(-[a-z0-9]+)*$")) and
  (.key == (.value.directory | gsub("-"; "_"))) and
  (.value.module == ("github.com/libops/" + .value.directory)) and
  (.value.repository == ("libops/" + .value.directory)) and
  (.value.module_version | test("^v(0|1)\\.[0-9]+\\.[0-9]+$")) and
  (if (.value.version | release_major) >= 2 then
     .value.module_version == "v0.0.0"
   else
     .value.module_version == .value.version
   end) and
  (.value.version | test("^v[0-9]+\\.[0-9]+\\.[0-9]+$")) and
  (.value.ref | test("^[0-9a-f]{40}$"))
)
