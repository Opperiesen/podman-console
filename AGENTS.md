# Podman Console — règles du dépôt

## Validation

Les bindings Podman utilisent les tags `containers_image_openpgp` et `remote` pour sélectionner
un backend OpenPGP pur Go et le mode service distant, ce qui évite `gpgme` et les APIs locales
non portables. Les commandes de référence sont :

```sh
go test -tags=containers_image_openpgp,remote ./...
go vet -tags=containers_image_openpgp,remote ./...
go build -tags=containers_image_openpgp,remote ./cmd/podman-console
```

Les tests par défaut sont déterministes et ne contactent jamais un hôte Podman. Toute validation
live doit être explicitement opt-in.

## Architecture

`internal/domain` porte les valeurs métier, `internal/podman` est le seul adaptateur des bindings,
`internal/app` orchestre l’état asynchrone et `internal/ui` ne connaît que les valeurs de domaine.
Les mutations sont confirmées dans le modèle avant l’appel transport, puis suivies d’un refresh
autoritaire.
