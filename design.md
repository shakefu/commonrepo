- Look for `./.commonreporc` # .reporc or .commonrepo.yml ?
    - Doesn't exist
        - Try to find repo root
        - Can't find one, error out
        - Find it, back to beginning
- Parse and store `.commonreporc`
- Look for `upstream:`
    - Doesn't exist, continue on
- Load each upstream
- For each upstream
    - Clone upstream repo@ref
    - Create Repo
    - Get list of all files
    - Process all files against renames
        - Allow for `upstream:` entry to map foobar/.commonreporc to `./.commonreporc`
    - Look for `.commonreporc`
        - Doesn't exist
            - Check if `upstream:` defines `include:`  # TBD: this is a pull model which is ... okayish?
                - Doesn't exist, throw away repo (no files included), maybe error, print warning?
    - Parse and store .commonreporc
    - Add `upstream:` include/exclude/renames to the end of the lists
    - Store `upstream:` template context
    - Add Repo to list of all repos
- For each Repo in list of all repos
    - Check for depth limit (default 3?)
    - Check for total # clones limit (default 10?)
    - Back to Look for `upstream:` with Repo
- All Repos have been cloned, .commonreporc parsed, upstreams found
- Reverse list of all Repos so we process most recently inherited last
- For each Repo in list of all repos
    - Filter list of all files against each `include:`
    - Filter list of all files against each `exclude:`
    - Create RepoFile for every filtered original filename
    - For each pattern in `template:`
        - Match RepoFile original filename
        - Mark RepoFile as templated
    - For each map in `rename:`
        - Match all RepoFile targets against pattern
        - If it matches, save the transformed target filename (multiple matches okay?)
    - Add RepoFile to master list of files
- For each RepoFile in the master list
    - Apply the template context if it's a template
    - TBD: Allow for YAML or JSON merge of configuration?
    - Write the file to the filesystem
        - Renames can move files out of repo root, e.g. `$HOME/blah`?
- For each Repo in list of all repos
    - For each tool in `installFrom:`
        - Create Installable
        - Add to master map of tools
            - Overwrite?
    - For each tool in `install:`
        - Add to master list of install targets
            - Overwrite versions/de-duplicate
- Determine available install managers?
    - TBD? This could be a simple command check, e.g. `apt-get --version`
    - Could be its own "tool" version script
    - How to update available install managers when they are tools in inherited repos to be installed?
- For each tool in master list of install targets
    - Check if we have an Installable
        - If not error/warn?
    - Check if version script exists and run it
        - If not warn?
    - If version is met, continue
    - Check available install manager scripts against Installable and `installWith:` option
    - Pick first match and install
- For each tool in master list of install targets
    - Check if version is met

File structure for Installables
```bash
./.commonrepo/install/  # `installFrom:`
    toolname/  # Arbitrary name for the tool
        version  # Platform independent script outputs just "1.0.0" style version
        apt  # Install manager specific scripts
        yum
        brew
        fish
        bin
        # ...
        ${GOOS}_${GOARCH}  # Platform arbitrary installer
```
