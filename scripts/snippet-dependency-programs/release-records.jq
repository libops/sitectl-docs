[to_entries[] | [.value.repository, .value.version, .value.ref]] as $records
| if ($records | length) != $expected_count then
    error("unexpected snippet dependency release record count")
  elif any($records[];
    (length != 3) or
    any(.[]; (type != "string") or (length == 0))
  ) then
    error("snippet dependency release records must contain three nonempty strings")
  else
    $records[] | @tsv
  end
