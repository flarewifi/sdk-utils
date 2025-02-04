# Git subtree for `sdk-utils`

## Add the remote url of `flarehotspot/sdk-utils`

```sh
git remote add sdk-utils git@github.com:flarehotspot/sdk-utils.git
```

## Split the utils to a `git subtree`.

```sh
git subtree split --prefix sdk/utils -b sdk-utils
```

This will create a new branch called `sdk-utils` which can be pushed to a git repo.

## Push the `sdk-utils` branch to a remote git repo.
```sh
git push sdk-utils sdk-utils:remote-branch-name
```

# Pushing changes to `sdk-utils`

```sh
# command guide
# git subtree push --prefix <utils dir name> <sdk-utils remote name or url> <desired local branch to push>
# don't worry, this will only push the changes inside the `utils` and not the entire local branch

# actual command
git subtree push --prefix sdk/utils sdk-utils development # or your desired local branch e.g. feat/utils-subtree
```

# Persist changes

For the changes to persist in other codebases that uses the go library, head over to the github or even to the local cloned repo of `sdk-utils` and create a git tag.

```sh
git checkout sdk-utils
git tag vx.x.x # creates a tag to the latest commit of the current branch
git push sdk-utils --tags # pushes the created tag
```

Then, update the `sdk-utils` library by specifying the version of the newly pushed tag.
```sh
go get -u github.com/flarehotspot/sdk-utils@vx.x.x
```

## Building `devkit`

Run the command: `make devkit`

Then you can find and test the devkit in `output/devkit` directory.
