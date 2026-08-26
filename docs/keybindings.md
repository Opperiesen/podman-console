# Raccourcis

## Inventaire

`↑/k` et `↓/j` déplacent la sélection, `Entrée` ouvre les détails, `r` actualise, `/` filtre,
`c` ouvre les profils, `l` affiche les logs, `m` affiche les métriques et `q` quitte.

## Conteneur

`s` démarre. `x` arrête, `R` redémarre et `D` supprime après confirmation exacte de la cible affichée.
`Échap` revient à la vue précédente.

## Images

Depuis l’inventaire, `i` ouvre les images locales. `↑/k` et `↓/j` déplacent la sélection, `Entrée`
ouvre les détails, `/` filtre localement, `P` ouvre le téléchargement et `D` demande la
confirmation de suppression exacte. `Échap` revient à l’inventaire des conteneurs.

Pendant un téléchargement, `Entrée` lance la référence saisie et `Échap` annule la requête tout
en conservant la progression reçue. Les identifiants de registre et les politiques d’authentification
restent gérés par Podman.

## Flux

Dans les logs, `f` active ou suspend le suivi et les flèches déplacent la fenêtre. Les données
déjà reçues restent visibles si le flux s’arrête.
