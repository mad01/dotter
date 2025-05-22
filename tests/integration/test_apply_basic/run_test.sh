#!/bin/bash
set -e # Exit immediately if a command exits with a non-zero status.

# Get the absolute path to the project root (assuming this script is in tests/integration/test_apply_basic)
PROJECT_ROOT=$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)
TEST_CASE_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

IMAGE_NAME="dotter-integration-test"

echo "Building Docker image ${IMAGE_NAME}..."
docker build -t ${IMAGE_NAME} ${PROJECT_ROOT} -f ${PROJECT_ROOT}/Dockerfile

echo "Running integration test for basic apply..."

# Mount the test-specific config and dotfiles source into the container
# Mount the project root as well, to allow dotter to potentially access other parts if needed (though ideally not for this test)
docker run --rm \
    -v "${TEST_CASE_DIR}/config.toml:/home/testuser/.config/dotter/config.toml" \
    -v "${TEST_CASE_DIR}/dotfiles_src:/home/testuser/dotfiles_src" \
    ${IMAGE_NAME} apply

# --- Verification --- 
# We run a new container to check the results. This is cleaner than trying to exec into the previous one.
echo "Verifying results..."

VERIFICATION_SCRIPT="
set -e
ls -la /home/testuser # Optional: for debugging

# Check .actual_bashrc symlink
if [ ! -L /home/testuser/.actual_bashrc ]; then
    echo 'ERROR: /home/testuser/.actual_bashrc is not a symlink!'
    exit 1
fi
LINK_TARGET_BASHRC=$(readlink /home/testuser/.actual_bashrc)
EXPECTED_TARGET_BASHRC='/home/testuser/dotfiles_src/.test_bashrc'
if [ \"${LINK_TARGET_BASHRC}\" != \"${EXPECTED_TARGET_BASHRC}\" ]; then
    echo \"ERROR: /home/testuser/.actual_bashrc points to ${LINK_TARGET_BASHRC}, expected ${EXPECTED_TARGET_BASHRC}\"
    exit 1
fi
echo '.actual_bashrc symlink OK'

# Check .actual_vimrc symlink
if [ ! -L /home/testuser/.actual_vimrc ]; then
    echo 'ERROR: /home/testuser/.actual_vimrc is not a symlink!'
    exit 1
fi
LINK_TARGET_VIMRC=$(readlink /home/testuser/.actual_vimrc)
EXPECTED_TARGET_VIMRC='/home/testuser/dotfiles_src/.test_vimrc'
if [ \"${LINK_TARGET_VIMRC}\" != \"${EXPECTED_TARGET_VIMRC}\" ]; then
    echo \"ERROR: /home/testuser/.actual_vimrc points to ${LINK_TARGET_VIMRC}, expected ${EXPECTED_TARGET_VIMRC}\"
    exit 1
fi
echo '.actual_vimrc symlink OK'

# Check content of linked .actual_bashrc
if ! grep -q 'TEST_VAR=\"bashrc_loaded\"' /home/testuser/.actual_bashrc; then
    echo 'ERROR: Content check failed for .actual_bashrc'
    exit 1
fi
echo 'Content of .actual_bashrc OK'

# Check content of linked .actual_vimrc
if ! grep -q 'set number' /home/testuser/.actual_vimrc; then
    echo 'ERROR: Content check failed for .actual_vimrc'
    exit 1
fi
echo 'Content of .actual_vimrc OK'


echo 'Verification successful!'
"

# Run the verification script in a new container with the same mounts (or just the target dir)
# For simplicity, we'll rely on the 'dotter apply' having created the necessary structure.
# A more robust approach might be to commit the container state from the apply step and then run verification against that committed image.
# Or, use a shared volume for the /home/testuser directory across the apply and verification containers.

# For now, we re-run dotter apply, then immediately check. This is not ideal as it re-runs the action.
# A better way: use a data volume or commit the container changes.

# Let's adjust: the first `docker run` creates the symlinks. 
# We need a way to inspect the state *after* that run.
# The simplest way without complex Docker volume management for this script is to exec into the running container,
# or to have the first container output some state that we can check on the host.
# For now, let's make the first container sleep after applying, then we can exec into it.
# No, that's too complex for an automated script.

# Let's try this: the `dotter apply` run will create the links. The next `docker run` will use the *same image*,
# but we need to ensure the *state* from the first run is visible.
# This is where named volumes come in handy or by committing the first container.

# Simpler approach for now: We will run 'dotter apply', then run a *new* container but mount the config
# and have it run 'dotter list' or execute the verification script directly.
# The key is that the *target* directories for symlinks are in /home/testuser which is ephemeral per container run.

# Correct approach for a test script like this: use a named volume to persist /home/testuser between runs.

VOLUME_NAME="dotter-test-home-$(date +%s)" # Unique volume name for this test run
docker volume create ${VOLUME_NAME} > /dev/null

# Run dotter apply, with /home/testuser mounted to the named volume
echo "Running dotter apply with persistent volume ${VOLUME_NAME}..."
docker run --rm \
    -v "${TEST_CASE_DIR}/config.toml:/home/testuser/.config/dotter/config.toml:ro" \
    -v "${TEST_CASE_DIR}/dotfiles_src:/home/testuser/dotfiles_src:ro" \
    -v "${VOLUME_NAME}:/home/testuser" \
    ${IMAGE_NAME} apply

echo "Verifying results in volume ${VOLUME_NAME}..."
# Now run the verification script in a new container, mounting the same named volume
docker run --rm \
    -v "${VOLUME_NAME}:/home/testuser" \
    -v "${TEST_CASE_DIR}/dotfiles_src:/home/testuser/dotfiles_src:ro" \
    ${IMAGE_NAME} /bin/sh -c "${VERIFICATION_SCRIPT}"

# Clean up the volume
echo "Cleaning up volume ${VOLUME_NAME}..."
docker volume rm ${VOLUME_NAME} > /dev/null

echo "Integration test completed successfully!" 