#/usr/bin/env -S bash

for i in $(seq 1 30); do
  curl http://localhost:8080/api/job -X PUT --json "{
  \"id\": $i,
  \"testname\": \"poop.sh.tmpl\",
  \"args\": {
    \"target\": \"$i\"
  }
}"
done
