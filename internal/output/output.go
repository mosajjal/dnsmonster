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

// Package output registers different output methods for dnsmonster.
// each output will register itself by running the init function.
// in the main package, if the output type is zero, the output will automatically
// be de-registered from dispatch. each output can provide its own specific flags
// and also benefit from generalflags using the `util` package
package output
// vim: foldmethod=marker
