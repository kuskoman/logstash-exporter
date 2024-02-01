#!/usr/bin/env bash

# execute: ./scripts/add_descriptions_to_readme.sh

scriptName="$(dirname "$0")/$(basename "$0")"

function getHelp() { # get descriptions and commands from Makefile
    i=0
    commands=()
    descriptions=()

    while read -r line; do
        if (( i % 2 == 0 ));
            then
                descriptions+=( "$(echo $line | sed 's/#:[ ]*//')" )
            else
                commands+=( $(echo "$line" | cut -d : -f 1) )
        fi

        ((i++))
    done < <(
        # https://stackoverflow.com/a/59087509
        grep -B1 -E "^[a-zA-Z0-9_-]+:([^\=]|$)" ./Makefile \
        | grep -v -- --
    )
}

FILE=README.md

getHelp

let startLine=$(grep -n "^#### Available Commands" $FILE | cut -d : -f 1)+2
let endLine=$(grep -n "^#### File Structure" $FILE | cut -d : -f 1)-2

# Updates "Available Commands" section:

if (( startLine <= endLine));
then
    $(sed -i "$startLine,${endLine}d" $FILE) # deletion of previous descriptions
fi

function printAvailableCommands() {
    curLine=$startLine
    stringToWrite="<!--- GENERATED by $scriptName --->"
    let commentLen=${#stringToWrite}-11
    i=0

    $(sed -i "${curLine}i\\${stringToWrite}" $FILE)
    let curLine++

    $(sed -i "${curLine}i\\ " $FILE)  # empty line
    let curLine++

    while (( $i < ${#commands[@]} ))
    do

        stringToWrite="- \`make ${commands[$i]}\`: ${descriptions[$i]}."
        $(sed -i "${curLine}i\\${stringToWrite}" $FILE)
        let curLine++

        let i++
    done

    $(sed -i "${curLine}i\\ " $FILE)  # empty line
    let curLine++

    stringToWrite="<!--- $( eval $( echo printf '"\*%.0s"' {1..$commentLen} ) ) --->" # multiple '*'
    $(sed -i "${curLine}i\\${stringToWrite}" $FILE)
    let curLine++

}

echo 'Updating "Available Commands" section...'

printAvailableCommands

# Updates "Example Usage" section:

let startLine=$(grep -n "^#### Example Usage" $FILE | cut -d : -f 1)+2
let endLine=$(grep -n "^## Helper Scripts" $FILE | cut -d : -f 1)-2

if (( startLine <= endLine));
then
    $(sed -i "$startLine,${endLine}d" $FILE) # deletion of previous descriptions
fi

function printExampleUsage() {
    curLine=$startLine
    stringToWrite="<!--- GENERATED by $scriptName --->"
    let commentLen=${#stringToWrite}-11
    i=0

    $(sed -i "${curLine}i\\${stringToWrite}" $FILE)
    let curLine++

    $(sed -i "${curLine}i\\ " $FILE)  # empty line
    let curLine++

    while (( $i < ${#commands[@]} ))
    do
        stringToWrite="${descriptions[$i]}:"
        $(sed -i "${curLine}i\\${stringToWrite}" $FILE)
        let curLine++

        $(sed -i "${curLine}i\\ " $FILE)
        let curLine++

        stringToWrite="    make ${commands[$i]}" # 4 spaces for tab (DON'T CHANGE IT)
        $(sed -i "${curLine}i\\${stringToWrite}" $FILE)
        let curLine++

        $(sed -i "${curLine}i\\ " $FILE)
        let curLine++

        let i++
    done

    stringToWrite="<!--- $( eval $( echo printf '"\*%.0s"' {1..$commentLen} ) ) --->" # multiple '*'
    $(sed -i "${curLine}i\\${stringToWrite}" $FILE)
    let curLine++

}

echo 'Updating "Example Usage" section...'

printExampleUsage

echo 'Done.'

