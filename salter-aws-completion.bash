_salter_aws_completion() {
    local cur prev opts actions
    COMPREPLY=()
    cur="${COMP_WORDS[COMP_CWORD]}"
    prev="${COMP_WORDS[COMP_CWORD-1]}"
    opts="-action -name -value -s -type -o -region -prefix -h"
    actions="get put put-from-template generate get-by-prefix"

    case "$prev" in
        -action)
            COMPREPLY=( $(compgen -W "$actions" -- "$cur") )
            return 0
            ;;
        -type)
            COMPREPLY=( $(compgen -W "string stringlist securestring" -- "$cur") )
            return 0
            ;;
    esac

    COMPREPLY=( $(compgen -W "$opts" -- "$cur") )
}
complete -F _salter_aws_completion salter-aws