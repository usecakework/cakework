# Releasing
To release to GitHub, you'll need to export a GITHUB_TOKEN environment variable, which should contain a valid GitHub token with the repo scope. It will be used to deploy releases to your GitHub repository.
`export GITHUB_TOKEN="YOUR_GH_TOKEN"`

To publish executable via Homebrew:
- Create new commit of local changes first
- 
- `VERSION=v1.0.65 && git tag -a $VERSION -m 'new revision' && git push origin $VERSION` (replace with new version)
- `git push` to trigger Github Actions to build a new revision

<!-- TODO: figure out how to install using brew -->
- `brew tap usecakework/cakeworkctl https://github.com/usecakework/homebrew-cakeworkctl`

Installing the cli:
# note this no longer works as the repo is no longer public
`curl -L https://raw.githubusercontent.com/usecakework/cakeworkctl/main/install.sh | sh`

Note: for developers of this package only: if you previously installed the cakework executable to the local go root for testing, delete that to test the install.sh script, i.e. `rm /Users/jessieyoung/go/bin/cakework`

Invoking the cli:
- `cakework run`

TODO: 
- include different versions of binaries for different architectures (currently only for Darwin arm64)
- maybe host the 3rd party executables elsewhere (too big, taking too long to do git pushes and pulls). Or look into git repos for large binaries 
- add instructions for testing cli locally (without having to tag and push each time)
- print out the output of each command as well as the error

To test building the CLI release locally:
`goreleaser build --rm-dist --snapshot`

If you want VSCode to be able to detect multiple modules, follow steps here: https://stackoverflow.com/questions/65748509/vscode-shows-an-error-when-having-multiple-go-projects-in-a-directory for updating the gopls setting