$2 ~ /\^\{\}$/ {
  peeled = $1
  next
}

{
  direct = $1
}

END {
  print peeled != "" ? peeled : direct
}
