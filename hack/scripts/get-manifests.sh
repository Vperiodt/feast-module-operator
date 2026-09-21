#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

COMPONENT_NAME="feastoperator"
SOURCE_PATH="infra/feast-operator/config"
DST_MANIFESTS_DIR="${PROJECT_ROOT}/config/manifests/${COMPONENT_NAME}"

if [[ "${ODH_PLATFORM_TYPE:-OpenDataHub}" == "OpenDataHub" ]]; then
    echo "Downloading manifests for ODH"
    REPO_URL="https://github.com/opendatahub-io/feast"
    COMMIT_SHA="586a5f3d5923cb44c55b5f7a4c2912585e3cef0c"
else
    echo "Downloading manifests for RHOAI"
    REPO_URL="https://github.com/red-hat-data-services/feast"
    COMMIT_SHA="de455fef0bc4a5680bab71645f065c9d086c8cb5"
fi

if [[ "${USE_LOCAL:-}" == "true" ]] && [[ -d "${PROJECT_ROOT}/../feast" ]]; then
    echo "Copying manifests from adjacent feast checkout"
    rm -rf "${DST_MANIFESTS_DIR}"
    mkdir -p "${DST_MANIFESTS_DIR}"
    cp -a "${PROJECT_ROOT}/../feast/${SOURCE_PATH}/." "${DST_MANIFESTS_DIR}/"
    echo "${COMMIT_SHA}" > "${DST_MANIFESTS_DIR}/.manifest-source-commit"
    echo "Manifests copied to ${DST_MANIFESTS_DIR}"
    exit 0
fi

MARKER_FILE="${DST_MANIFESTS_DIR}/.manifest-source-commit"
if [[ "${FORCE_GET_MANIFESTS:-}" != "true" && -f "${DST_MANIFESTS_DIR}/manager/manager.yaml" ]]; then
    if [[ ! -f "${MARKER_FILE}" ]]; then
        echo "${COMMIT_SHA}" > "${MARKER_FILE}"
    fi
    echo "Bundled manifests present under ${DST_MANIFESTS_DIR}, skipping download"
    echo "(set FORCE_GET_MANIFESTS=true to re-fetch from GitHub)"
    exit 0
fi

if [[ "${FORCE_GET_MANIFESTS:-}" != "true" && -f "${MARKER_FILE}" ]]; then
    if [[ "$(tr -d '[:space:]' < "${MARKER_FILE}")" == "${COMMIT_SHA}" ]]; then
        echo "Manifests already at ${COMMIT_SHA}, skipping download (set FORCE_GET_MANIFESTS=true to refresh)"
        exit 0
    fi
fi

TMP_DIR=$(mktemp -d -t "odh-feast-manifests.XXXXXXXXXX")
trap 'rm -rf -- "${TMP_DIR}"' EXIT

git_fetch_with_retry() {
    local attempt=1 max_attempts=3
    while [[ "${attempt}" -le "${max_attempts}" ]]; do
        if git -C "${TMP_DIR}" fetch --depth 1 origin "${COMMIT_SHA}"; then
            return 0
        fi
        echo "git fetch failed (attempt ${attempt}/${max_attempts})" >&2
        if [[ "${attempt}" -lt "${max_attempts}" ]]; then
            sleep $((attempt * 5))
        fi
        attempt=$((attempt + 1))
    done
    return 1
}

echo "Fetching ${REPO_URL}@${COMMIT_SHA} ..."
echo "(shallow git clone — usually 30-90s; retries on network blips)"

git -C "${TMP_DIR}" init -q
git -C "${TMP_DIR}" remote add origin "${REPO_URL}"
if ! git_fetch_with_retry; then
    echo "ERROR: could not fetch manifests from GitHub after 3 attempts." >&2
    echo "  - Bundled manifests already in repo? Run: SKIP_GET_MANIFESTS=1 make deploy-openshift" >&2
    echo "  - Or use local feast checkout: USE_LOCAL=true ./hack/scripts/get-manifests.sh" >&2
    exit 1
fi
git -C "${TMP_DIR}" reset -q --hard "${COMMIT_SHA}"

rm -rf "${DST_MANIFESTS_DIR}"
mkdir -p "${DST_MANIFESTS_DIR}"
cp -a "${TMP_DIR}/${SOURCE_PATH}/." "${DST_MANIFESTS_DIR}/"
echo "${COMMIT_SHA}" > "${MARKER_FILE}"

echo "Manifests downloaded to ${DST_MANIFESTS_DIR}"
