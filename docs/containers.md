# Créer un conteneur depuis une image locale

Depuis l’inventaire des images (`i`), sélectionnez une image déjà présente sur la cible Podman,
puis appuyez sur `n`. L’image est capturée avec son identifiant complet : aucun téléchargement et
aucune référence distante implicite ne sont ajoutés.

Le formulaire demande :

- un nom explicite de 1 à 63 caractères, commençant par une lettre ou un chiffre, puis composé de
  lettres, chiffres, `.`, `_` ou `-` ;
- une commande optionnelle saisie comme une suite d’arguments (`sleep 60`, par exemple).

La commande n’est jamais exécutée par un shell. Les opérateurs, substitutions, guillemets et
antislashs sont refusés ; si le champ est vide, Podman utilise l’entrypoint et la commande définis
par l’image.

La confirmation affiche la cible active, la référence d’image, l’identifiant complet, le nom et la
commande effective. Après validation, Podman reçoit exactement une création détachée, puis un
démarrage. Les inventaires des conteneurs et des images sont relus depuis l’hôte avant d’annoncer
la fin du workflow.

## Annulation et résultat partiel

`Échap` annule le formulaire ou la confirmation sans envoyer de mutation. Une sélection d’image ou
une cible devenue obsolète est refusée avant la création.

La création et le démarrage ne forment pas une transaction atomique. Si Podman renvoie un
identifiant mais refuse ou perd le démarrage, l’interface affiche **créé mais non démarré** avec
l’identifiant exact, actualise les deux inventaires et ne supprime ni ne relance automatiquement le
conteneur. Il reste alors disponible dans le workflow normal pour inspection, arrêt ou suppression.

Les identifiants, certificats, politiques de signature et authentifications restent gérés par le
service Podman configuré ; Podman Console ne collecte ni mot de passe ni clé privée.
