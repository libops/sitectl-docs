[.Require[]? | select(.Path | test($namespace_pattern))] as $requires
| if any($requires[]; (.Path | test($pattern) | not)) then
    error("generator contains an unsupported LibOps sitectl require module path")
  else
    [$requires[] | {Path, Version, Indirect: (.Indirect // false)}]
    | sort_by(.Path)
  end
