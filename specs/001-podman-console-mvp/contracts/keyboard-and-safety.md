# Keyboard and Safety Contract

## Global bindings

| Key | Action |
|---|---|
| `q`, `Ctrl+C` | Quit unless a confirmation or text field is active |
| `?` | Toggle help |
| `r` | Refresh the active inventory or detail view |
| `c` | Open connection selector |
| `Esc` | Cancel dialog, stream, or pending action |
| `j`/`Down` | Move selection down |
| `k`/`Up` | Move selection up |
| `Enter` | Open the selected row or accept a focused dialog |

## Container actions

| Key | Action | Confirmation |
|---|---|---|
| `s` | Start selected container | No, if the host reports it is stopped |
| `x` | Stop selected container | Yes |
| `R` | Restart selected container | Yes |
| `D` | Remove selected container | Yes |
| `l` | Open logs | No |
| `m` | Open metrics | No |

## Confirmation requirements

Before a stop, restart, or remove request is sent, the dialog MUST show:

- the action name;
- the active host name;
- the container name;
- the complete container identifier or an unambiguous suffix plus a way to inspect the full ID.

`Enter` confirms and `Esc` cancels. Cancellation sends no mutation request. A confirmation is
invalidated if the active host or selected container changes before submission.
