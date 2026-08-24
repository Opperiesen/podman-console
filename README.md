# Podman Console

Podman Console est une interface terminal indépendante pour administrer une cible Podman locale
ou distante : inventaire, détails, cycle de vie, logs suivis et métriques CPU/mémoire.

## Développement

```sh
go test -tags=containers_image_openpgp,remote ./...
go vet -tags=containers_image_openpgp,remote ./...
go build -tags=containers_image_openpgp,remote ./cmd/podman-console
```

Les tags de build sélectionnent le backend OpenPGP pur Go des bindings Podman et leur mode
service distant : ils évitent une dépendance native `gpgme` et rendent le binaire reproductible
sur macOS, Linux et Windows.

Le test par défaut n’exige aucun hôte Podman. Les profils sont stockés dans le répertoire de
configuration utilisateur, sans mot de passe ni clé privée.

## Périmètre MVP

Une seule cible est active à la fois. Les opérations en masse, la construction d’images, les
registries, l’orchestration de pods et l’agrégation de plusieurs hôtes ne font pas partie de ce
MVP.
