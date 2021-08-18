package main

var BASH_COMPLETION_TEMPLATE = `
_completion_dnsmonster() {
    # All arguments except the first one
    args=("${COMP_WORDS[@]:1:$COMP_CWORD}")

    # Only split on newlines
    local IFS=$'\n'

    # Call completion (note that the first element of COMP_WORDS is
    # the executable itself)
    COMPREPLY=($(GO_FLAGS_COMPLETION=verbose ${COMP_WORDS[0]} "${args[@]}"))
    return 0
}

complete -F _completion_dnsmonster dnsmonster
`
