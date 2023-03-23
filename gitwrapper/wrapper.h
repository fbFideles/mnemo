#include <git2/global.h>

void git_init() { git_libgit2_init(); }

void git_shutdown() { git_libgit2_shutdown(); }
