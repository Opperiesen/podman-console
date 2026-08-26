# Raccourcis

## Inventaire

`↑/k` et `↓/j` déplacent la sélection, `Entrée` ouvre les détails, `r` actualise, `/` filtre,
`c` ouvre les profils, `l` affiche les logs, `m` affiche les métriques et `q` quitte.

## Conteneur

`s` démarre. `x` arrête, `R` redémarre et `D` supprime après confirmation exacte de la cible affichée.
`Échap` revient à la vue précédente.

## Images

Depuis l’inventaire, `i` ouvre les images locales. `↑/k` et `↓/j` déplacent la sélection, `Entrée`
ouvre les détails, `n` ouvre la création d’un conteneur, `/` filtre localement, `P` ouvre le
téléchargement et `D` demande la confirmation de suppression exacte. `Échap` revient à
l’inventaire des conteneurs.

Pendant un téléchargement, `Entrée` lance la référence saisie et `Échap` annule la requête tout
en conservant la progression reçue. Les identifiants de registre et les politiques d’authentification
restent gérés par Podman.

## Création depuis une image locale

Depuis une image sélectionnée, `n` ouvre le formulaire. Le premier champ reçoit le nom du
conteneur ; `Tab` passe à la commande optionnelle, qui doit contenir des arguments uniquement.
`Entrée` affiche la confirmation exacte, puis `Entrée` ou `y` crée et démarre le conteneur en
mode détaché. `Échap` annule avant mutation ; pendant la requête, il demande son annulation et
l’état éventuellement partiel reste affiché.

## Flux

Dans les logs, `f` active ou suspend le suivi et les flèches déplacent la fenêtre. Les données
déjà reçues restent visibles si le flux s’arrête.
