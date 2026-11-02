#!/usr/bin/env bash

# Copyright 2026 Chainguard, Inc.
# SPDX-License-Identifier: Apache-2.0

# Runs import-github-app-key against a stub gcloud (and a stub sleep, so
# polling costs nothing) with real openssl and throwaway keys. No cloud
# resources are touched.

set -o errexit
set -o nounset
set -o pipefail

script="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/import-github-app-key"
work=$(mktemp -d)
trap 'rm -rf "${work}"' EXIT

readonly key_name=projects/proj-123/locations/global/keyRings/ring/cryptoKeys/app-key

mkdir -p "${work}/bin" "${work}/keys"
# GitHub serves App keys as PKCS#1 ("BEGIN RSA PRIVATE KEY"). OpenSSL 3 needs
# -traditional to write that; LibreSSL writes it by default and lacks the flag.
openssl genrsa -traditional -out "${work}/keys/rsa2048.pem" 2048 2>/dev/null ||
  openssl genrsa -out "${work}/keys/rsa2048.pem" 2048 2>/dev/null
openssl genrsa -out "${work}/keys/other2048.pem" 2048 2>/dev/null
openssl genrsa -out "${work}/keys/rsa4096.pem" 4096 2>/dev/null
openssl ecparam -genkey -name prime256v1 -noout -out "${work}/keys/ec.pem" 2>/dev/null
openssl rsa -in "${work}/keys/rsa2048.pem" -pubout -out "${work}/keys/public.pem" 2>/dev/null
# The fingerprint GitHub lists for rsa2048.pem, computed independently of the
# script.
want_fingerprint=$(openssl rsa -in "${work}/keys/rsa2048.pem" -pubout -outform DER 2>/dev/null | openssl sha256 -binary | base64)

# The stub gcloud answers the calls the script makes, driven by files in
# ${STUB}: key-info (the describe line), no-cryptography, import-fails,
# import-state (the version's state once it leaves PENDING_IMPORT),
# public-key (overrides the public key KMS reports) and public-key-fails. It logs each call to
# ${STUB}/calls. Like KMS, the import job and the version each report a
# pending state once before they are usable, and refuse use until then; the
# version it imports holds the public half of the DER key it was given; and
# version numbers continue from ${STUB}/versions.
cat >"${work}/bin/gcloud" <<'EOF'
#!/usr/bin/env bash
set -o errexit -o nounset -o pipefail
echo "$*" >>"${STUB}/calls"
flag() {
  local a
  for a in "${args[@]}"; do
    [[ "${a}" != "--$1="* ]] || echo "${a#*=}"
  done
}
args=("$@")
case "$*" in
"info --format=value(basic.python_location)")
  echo "${STUB}/bin/gcloud-python"
  ;;
"kms keys describe "*)
  cat "${STUB}/key-info"
  ;;
"kms import-jobs create "*)
  echo "${4}" >>"${STUB}/jobs"
  ;;
"kms import-jobs describe "*)
  if [[ ! -e "${STUB}/job-active" ]]; then
    : >"${STUB}/job-active"
    echo PENDING_GENERATION
  else
    echo ACTIVE
  fi
  ;;
"kms keys versions import "*)
  der=$(flag target-key-file)
  echo "${der}" >"${STUB}/der-path"
  echo "sitepackages=${CLOUDSDK_PYTHON_SITEPACKAGES:-}" >"${STUB}/import-env"
  if [[ ! -e "${STUB}/job-active" ]]; then
    echo "ERROR: (gcloud.kms.keys.versions.import) FAILED_PRECONDITION: import job is not active" >&2
    exit 1
  fi
  if [[ -e "${STUB}/import-fails" ]]; then
    echo "ERROR: (gcloud.kms.keys.versions.import) PERMISSION_DENIED" >&2
    exit 1
  fi
  openssl pkey -inform DER -in "${der}" -pubout -out "${STUB}/imported.pub" 2>/dev/null
  n=$(($(cat "${STUB}/versions") + 1))
  echo "${n}" >"${STUB}/versions"
  echo "projects/$(flag project)/locations/$(flag location)/keyRings/$(flag keyring)/cryptoKeys/$(flag key)/cryptoKeyVersions/${n}"
  ;;
"kms keys versions describe "*"--format=value(state)")
  if [[ ! -e "${STUB}/version-polled" ]]; then
    : >"${STUB}/version-polled"
    echo PENDING_IMPORT
  else
    cp "${STUB}/import-state" "${STUB}/version-state"
    cat "${STUB}/version-state"
  fi
  ;;
"kms keys versions describe "*"--format=value(importFailureReason)")
  echo "wrapped key is malformed"
  ;;
"kms keys versions get-public-key "*)
  if [[ "$(cat "${STUB}/version-state" 2>/dev/null)" != ENABLED ]]; then
    echo "ERROR: (gcloud.kms.keys.versions.get-public-key) FAILED_PRECONDITION: version is not enabled" >&2
    exit 1
  fi
  if [[ -e "${STUB}/public-key-fails" ]]; then
    echo "ERROR: (gcloud.kms.keys.versions.get-public-key) UNAVAILABLE" >&2
    exit 1
  fi
  if [[ -e "${STUB}/public-key" ]]; then
    cat "${STUB}/public-key"
  else
    cat "${STUB}/imported.pub"
  fi
  ;;
*)
  echo "stub gcloud: unexpected call: $*" >&2
  exit 99
  ;;
esac
EOF
cat >"${work}/bin/gcloud-python" <<'EOF'
#!/usr/bin/env bash
[[ "$*" == "-c import cryptography" && ! -e "${STUB}/no-cryptography" ]]
EOF
cat >"${work}/bin/sleep" <<'EOF'
#!/usr/bin/env bash
echo "$1" >>"${STUB}/sleeps"
EOF
# A shred that cannot delete the file named in ${STUB}/undeletable, as when
# the PEM's directory is read-only. Put it on PATH only for the runs that
# need it, so the others use the real shred.
mkdir -p "${work}/shred-bin"
cat >"${work}/shred-bin/shred" <<'EOF'
#!/usr/bin/env bash
file=${!#}
if [[ -e "${STUB}/undeletable" && "${file}" == "$(cat "${STUB}/undeletable")" ]]; then
  echo "shred: ${file}: failed to open for writing: Permission denied" >&2
  exit 1
fi
rm -f -- "${file}"
EOF
chmod +x "${work}/bin/gcloud" "${work}/bin/gcloud-python" "${work}/bin/sleep" "${work}/shred-bin/shred"

export STUB="${work}/stub"
reset_stub() {
  rm -rf "${STUB}" "${work}/tmp" "${work}/pem"
  mkdir -p "${STUB}/bin" "${work}/tmp"
  cp "${work}/bin/gcloud-python" "${STUB}/bin/"
  echo "ASYMMETRIC_SIGN,True,RSA_SIGN_PKCS1_2048_SHA256,SOFTWARE" >"${STUB}/key-info"
  echo ENABLED >"${STUB}/import-state"
  echo 1 >"${STUB}/versions"
  : >"${STUB}/calls"
  cp "${work}/keys/${1:-rsa2048}.pem" "${work}/pem"
}

failures=0
fail() {
  echo "FAIL: $*" >&2
  failures=$((failures + 1))
}

# run [KEY_NAME] runs the script on ${work}/pem, setting status, out and err.
run() {
  status=0
  PATH="${work}/bin:${PATH}" TMPDIR="${work}/tmp" \
    "${script}" "${work}/pem" "${1:-${key_name}}" >"${work}/out" 2>"${work}/err" </dev/null || status=$?
  out=$(cat "${work}/out")
  err=$(cat "${work}/err")
}

# expect_status NAME WANT
expect_status() {
  [[ "${status}" == "$2" ]] || fail "$1: exit ${status}, want $2"$'\n'"${out}"$'\n'"${err}"
}

expect_pem_kept() {
  [[ -f "${work}/pem" ]] || fail "$1: the PEM was deleted"
  grep -q "kept ${work}/pem" <<<"${err}" || fail "$1: no word that the PEM was kept"$'\n'"${err}"
}

# expect_version_reported NAME VERSION: the run failed after KMS created
# VERSION, so a re-run would add a version beside one that may hold the key.
expect_version_reported() {
  grep -qF "KMS already has ${key_name}/cryptoKeyVersions/$2" <<<"${err}" || fail "$1: no word that version $2 exists"$'\n'"${err}"
  grep -qF "SHA256:${want_fingerprint}" <<<"${err}" || fail "$1: no fingerprint to check version $2 against"$'\n'"${err}"
  ! grep -q "re-run to retry" <<<"${err}" || fail "$1: told the operator to retry"$'\n'"${err}"
}

expect_no_kms_calls() {
  ! grep -q '^kms ' "${STUB}/calls" || fail "$1: called KMS:"$'\n'"$(cat "${STUB}/calls")"
}

expect_no_import_job() {
  [[ ! -e "${STUB}/jobs" ]] || fail "$1: created an import job"
}

expect_tmp_empty() {
  [[ -z "$(ls -A "${work}/tmp")" ]] || fail "$1: left files in TMPDIR: $(ls -A "${work}/tmp")"
}

# Happy path: the operator learns which version to reference and can check
# the key against GitHub before trusting it.
name="a successful import reports the version Terraform must reference and the fingerprint GitHub lists"
reset_stub
run
expect_status "${name}" 0
grep -qx "key version: 2" <<<"${out}" || fail "${name}: no version line"$'\n'"${out}"
grep -qx "fingerprint: SHA256:${want_fingerprint}" <<<"${out}" || fail "${name}: want fingerprint SHA256:${want_fingerprint}"$'\n'"${out}"
grep -qF "${key_name}/cryptoKeyVersions/2" <<<"${out}" || fail "${name}: no full version name"$'\n'"${out}"

name="a successful import deletes the downloaded PEM, so the KMS key is the only copy"
[[ ! -e "${work}/pem" ]] || fail "${name}: PEM still exists"

name="a successful import leaves no DER copy of the key behind"
[[ -n "$(cat "${STUB}/der-path")" ]] || fail "${name}: import got no key file"
expect_tmp_empty "${name}"

name="the key resource name supplies every KMS flag, so no per-stack flag can be mistyped"
grep -q -- "kms keys versions import --key=app-key --project=proj-123 --location=global --keyring=ring " "${STUB}/calls" ||
  fail "${name}: import call lacks the parsed flags"$'\n'"$(cat "${STUB}/calls")"
grep -q -- "kms import-jobs create app-key-[0-9a-z-]* --project=proj-123 --location=global --keyring=ring --import-method=rsa-oaep-4096-sha256-aes-256 --protection-level=software" "${STUB}/calls" ||
  fail "${name}: import job call lacks the parsed flags"$'\n'"$(cat "${STUB}/calls")"

name="gcloud's import sees user site-packages, where pip --user puts cryptography"
grep -qx "sitepackages=1" "${STUB}/import-env" || fail "${name}: $(cat "${STUB}/import-env")"


name="the script prints no key material"
! grep -q "PRIVATE KEY" <<<"${out}${err}" || fail "${name}: output contains a private key"

# Operators often chmod 400 a downloaded private key. The PEM is deleted after
# KMS holds the key, so failing there would hide the version Terraform needs.
name="a read-only PEM is deleted after the import, and the run reports the version"
reset_stub
chmod 400 "${work}/pem"
run
expect_status "${name}" 0
grep -qx "key version: 2" <<<"${out}" || fail "${name}: no version line"$'\n'"${out}"
[[ ! -e "${work}/pem" ]] || fail "${name}: PEM still exists"

# Once KMS holds the key, a re-run would import yet another version, so a PEM
# that cannot be deleted is left for the operator to remove.
name="a PEM that cannot be deleted does not fail the import, so the operator learns the version instead of importing again"
reset_stub
echo "${work}/pem" >"${STUB}/undeletable"
PATH="${work}/shred-bin:${PATH}" run
expect_status "${name}" 0
grep -qx "key version: 2" <<<"${out}" || fail "${name}: no version line"$'\n'"${out}"
grep -qF "could not delete ${work}/pem; delete it by hand" <<<"${err}" || fail "${name}: no word to delete the PEM by hand"$'\n'"${err}"
! grep -q "re-run to retry" <<<"${err}" || fail "${name}: told the operator to retry"$'\n'"${err}"

# Rotation imports a second key into the same key ring. Import jobs cannot be
# reused once expired, so a fixed job name would fail the second time.
name="each run creates its own import job, so a rotation never collides with an old job"
reset_stub
run
cp "${work}/keys/other2048.pem" "${work}/pem"
run
expect_status "${name}" 0
grep -qx "key version: 3" <<<"${out}" || fail "${name}: second import is not version 3"$'\n'"${out}"
(($(sort -u "${STUB}/jobs" | wc -l) == 2)) || fail "${name}: job names: $(cat "${STUB}/jobs")"

# A wrong argument fails before anything touches KMS.
for bad in \
  "gcpkms://${key_name}" \
  "${key_name}/cryptoKeyVersions/1" \
  projects/proj-123/locations/global/keyRings/ring \
  "projects/proj-123/locations/global/keyRings/ring/cryptoKeys/" \
  app-key; do
  name="${bad} is refused as a key resource name before gcloud runs"
  reset_stub
  run "${bad}"
  expect_status "${name}" 1
  grep -q "not a key resource name" <<<"${err}" || fail "${name}: ${err}"
  [[ ! -s "${STUB}/calls" ]] || fail "${name}: ran gcloud"
  [[ -f "${work}/pem" ]] || fail "${name}: the PEM was deleted"
done

# GitHub App keys are RSA-2048; anything else is the wrong file and would
# leave the key unusable for RSA_SIGN_PKCS1_2048_SHA256.
for bad in ec rsa4096 public; do
  name="a ${bad} PEM is refused before KMS is touched, because GitHub App keys are 2048-bit RSA private keys"
  reset_stub "${bad}"
  run
  expect_status "${name}" 1
  expect_no_kms_calls "${name}"
  [[ -f "${work}/pem" ]] || fail "${name}: the PEM was deleted"
done

name="missing cryptography fails before an import job exists and names the pip command that fixes it"
reset_stub
: >"${STUB}/no-cryptography"
run
expect_status "${name}" 1
expect_no_kms_calls "${name}"
grep -qF "\"${STUB}/bin/gcloud-python\" -m pip install --user cryptography" <<<"${err}" || fail "${name}: ${err}"

for info in \
  "ASYMMETRIC_SIGN,,RSA_SIGN_PKCS1_2048_SHA256,SOFTWARE" \
  "ASYMMETRIC_SIGN,True,RSA_SIGN_PKCS1_4096_SHA256,SOFTWARE" \
  "ENCRYPT_DECRYPT,True,GOOGLE_SYMMETRIC_ENCRYPTION,SOFTWARE"; do
  name="a key described as ${info} is refused before an import job exists, since the App could never sign with it"
  reset_stub
  echo "${info}" >"${STUB}/key-info"
  run
  expect_status "${name}" 1
  expect_no_import_job "${name}"
done

name="a failed import removes the DER copy but keeps the PEM, so the operator can retry"
reset_stub
: >"${STUB}/import-fails"
run
expect_status "${name}" 1
[[ -n "$(cat "${STUB}/der-path")" ]] || fail "${name}: import got no key file"
[[ ! -e "$(cat "${STUB}/der-path")" ]] || fail "${name}: DER file survived"
expect_tmp_empty "${name}"
expect_pem_kept "${name}"

# A version that failed to import holds no key, so retrying is safe.
name="IMPORT_FAILED surfaces KMS's reason and keeps the PEM for a retry"
reset_stub
echo IMPORT_FAILED >"${STUB}/import-state"
run
expect_status "${name}" 1
grep -q "wrapped key is malformed" <<<"${err}" || fail "${name}: ${err}"
expect_tmp_empty "${name}"
expect_pem_kept "${name}"
grep -q "re-run to retry" <<<"${err}" || fail "${name}: no word to retry"$'\n'"${err}"

# Once KMS has created a version, it may hold the key even though the run
# failed; a re-run would import a second version of the same key.
name="a failure to read back the public key names the version KMS created instead of advising a re-run"
reset_stub
: >"${STUB}/public-key-fails"
run
expect_status "${name}" 1
expect_pem_kept "${name}"
expect_version_reported "${name}" 2

name="a version still PENDING_IMPORT when polling gives up is named instead of advising a re-run"
reset_stub
echo PENDING_IMPORT >"${STUB}/import-state"
run
expect_status "${name}" 1
expect_pem_kept "${name}"
expect_version_reported "${name}" 2

# Pointing Terraform at a version holding some other key would break every
# token mint, so a mismatch must stop the run with the PEM still on disk.
name="a version whose public key differs from the PEM fails and keeps the PEM"
reset_stub
openssl rsa -in "${work}/keys/other2048.pem" -pubout -out "${STUB}/public-key" 2>/dev/null
run
expect_status "${name}" 1
grep -q "holds public key" <<<"${err}" || fail "${name}: ${err}"
expect_pem_kept "${name}"

if ((failures > 0)); then
  echo "${failures} check(s) failed" >&2
  exit 1
fi
echo "ok"
