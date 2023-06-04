/* {{{ Copyright (C) 2022 Ali Mosajjal
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <https://www.gnu.org/licenses/>. }}} */

package util

const bashCompletionTemplate = `
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

const systemdServiceTemplate = `
[Unit]
Description=dnsmonster service

[Service]
Type=simple
Restart=always
RestartSec=3
ExecStart=/usr/bin/dnsmonster --config /etc/dnsmonster.ini

[Install]
WantedBy=multi-user.target
`
// vim: foldmethod=marker
