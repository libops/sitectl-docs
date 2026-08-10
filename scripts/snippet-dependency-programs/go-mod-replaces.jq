[.Replace[]? | select(.Old.Path | test($namespace_pattern))] as $replaces
| if any($replaces[]; (.Old.Path | test($pattern) | not)) then
    error("generator contains an unsupported LibOps sitectl replace module path")
  else
    [$replaces[] | {
      Path: .Old.Path,
      OldVersion: (.Old.Version // ""),
      NewPath: (.New.Path // ""),
      NewVersion: (.New.Version // "")
    }] | sort_by(.Path)
  end
