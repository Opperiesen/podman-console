# Connexions Podman

Podman Console conserve uniquement le nom d’une cible, son URI de service et, si nécessaire,
le chemin d’un fichier d’identité. Les mots de passe et clés privées restent dans l’agent SSH ou
le trousseau du système.

## URI acceptées

- `unix:///run/user/1000/podman/podman.sock` pour un socket local Linux ;
- `unix:///Users/<user>/.local/share/containers/podman/machine/podman.sock` pour un socket local
  exposé par une machine Podman macOS ;
- `ssh://user@host/run/user/1000/podman/podman.sock` pour un service distant ;
- `tcp://host:port` pour un service TCP explicitement sécurisé par l’environnement.

La touche `c` ouvre le sélecteur (`p` reste acceptée). `n` crée un profil, `e` modifie la cible sélectionnée et `D`
la retire. La sélection active est écrite dans le fichier de configuration de l’utilisateur.

## Dépannage

Une cible injoignable est signalée séparément d’un refus d’autorisation. Vérifier l’URI, le
service Podman, l’agent SSH et les droits du socket avant de relancer. Une mutation refusée n’est
jamais répétée automatiquement.
