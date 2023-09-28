#!/bin/bash

version="$1"
if [ -z $version ]; then
    echo "Please specify version:"
    exit -1
fi

version_file="./miso/version.go"
printf "Releasing verion $version\n"


printf "1. Writing version file $version_file\n"
> $version_file
printf "package miso\n\n" >> $version_file
printf "const (\n\tMisoVersion = \"%s\"\n" $version >> $version_file
printf ")\n" >> $version_file
if [ "$?" -ne 0 ]; then
    exit -1
fi
echo "Finished writing version file"
cat $version_file
echo

echo "2. Creating commit for release"
git commit -am "Release $version"
if [ "$?" -ne 0 ]; then
    exit -1
fi
echo "Created commit for release"
show=$(git show)
printf "\n>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>\n$show\n<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<\n"
echo

echo "3. Creating git tag for release"
git tag "$version"
echo "Git tagged version $version"
echo

if [ "$?" -ne 0 ]; then
    exit -1
fi

echo "Done, it's time to push your tag to remote origin! :D"

echo "git push && git push origin $version;"
