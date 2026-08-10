[to_entries[] | [.key, .value.directory, .value.ref]] as $records
| if ($records | length) != $expected_count then
    error("unexpected snippet dependency checkout record count")
  elif any($records[];
    (length != 3) or
    any(.[]; (type != "string") or (length == 0))
  ) then
    error("snippet dependency checkout records must contain three nonempty strings")
  else
    $records[] | @tsv
  end
