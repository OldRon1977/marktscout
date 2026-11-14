#!/bin/sh
# Sucht nach Daten, die einen Nutzer verraten: E-Mail-Adressen, Passwörter,
# Sitzungscookies, Schlüssel und Tokens, Browserprofile, Laufzeitdaten.
#
# Aufruf:
#   scripts/datenschutz.sh --staged          gestagte Änderungen und Autor (Pre-Commit-Hook)
#   scripts/datenschutz.sh --nachricht DATEI Commit-Nachricht (Commit-Msg-Hook)
#   scripts/datenschutz.sh --alle            alle versionierten Dateien und alle Commits (CI)
#
# Ausgabe: je Fund Ort und Art, nie der gefundene Wert. Exit 1 bei Fund.
set -u

modus="${1:---staged}"
cd "$(git rev-parse --show-toplevel)" || exit 2

# Dateien, die nie ins Repository gehören
pfade='(^|/)(browser-profil|\.auth|daten)/|storage-state[^/]*\.json$|(^|/)(cookies?(\.txt|\.json|\.sqlite)?|Cookies|Login Data)$|(^|/)\.env(\.[^/]*)?$|(^|/)zugangsdaten[^/]*$|\.(db|db-wal|db-shm|db-journal|har|pem|key|p12|pfx|kdbx)$'

# Inhalte: Art|erweiterter regulärer Ausdruck
muster='E-Mail-Adresse|[A-Za-z0-9._%+-]+@[A-Za-z0-9-]+(\.[A-Za-z0-9-]+)*\.[A-Za-z]{2,}
Passwort|([Pp][Aa][Ss][Ss][Ww][Oo][Rr][DdTt]|[Kk]ennwort|[Pp][Ww][Dd])["'"'"']?[[:space:]]*:?[:=][[:space:]]*["'"'"']?[^[:space:]"'"'"',;]{4,}
Cookie|([Ss]et-)?[Cc]ookie:[[:space:]]*[^[:space:]=]+=
Sitzungs-ID oder Token|([Ss]ession|[Ss]ess|[Ss]id|[Tt]oken|[Aa]uth)[_-]?([Ii][Dd]|[Kk]ey)?["'"'"']?[[:space:]]*:?[:=][[:space:]]*["'"'"']?[A-Za-z0-9_-]{16,}
Privater Schlüssel|-----BEGIN [A-Z ]*PRIVATE KEY-----
API-Schlüssel|(ghp_[A-Za-z0-9]{30,}|github_pat_[A-Za-z0-9_]{30,}|AIza[0-9A-Za-z_-]{30,}|sk-[A-Za-z0-9_-]{20,}|xox[abp]-[A-Za-z0-9-]{10,})
Bearer-Token|[Bb]earer[[:space:]]+[A-Za-z0-9._-]{20,}'

# Erlaubte E-Mail-Adressen in Inhalten und Nachrichten: Beispieldomains und GitHub-noreply
erlaubt='@(example\.(com|org|net)|[A-Za-z0-9.-]+\.(invalid|test|example)|users\.noreply\.github\.com)$'

# Erlaubte Autor- und Committer-Adressen: nur GitHub-noreply
autor_erlaubt='^([0-9]+\+)?[A-Za-z0-9._-]+(\[bot\])?@users\.noreply\.github\.com$|^noreply@github\.com$'

# Diese Datei enthält die Muster selbst.
selbst='scripts/datenschutz.sh'

fund=0
melde() {
	echo "datenschutz: $1" >&2
	fund=1
}

# pruefe_inhalt: liest "ort<TAB>zeile" aus $inhalt und meldet Funde je Ort.
pruefe_inhalt() {
	alt_ifs=$IFS
	IFS='
'
	for zeile in $muster; do
		art=${zeile%%|*}
		re=${zeile#*|}
		if [ "$art" = "E-Mail-Adresse" ]; then
			# jede Adresse einzeln gegen die erlaubten Domains prüfen
			orte=$(printf '%s\n' "$inhalt" | grep -v "^$selbst	" | grep -E -- "$re" |
				RE="$re" OK="$erlaubt" awk -F '\t' '{
					re = ENVIRON["RE"]; ok = ENVIRON["OK"]
					s = $2
					while (match(s, re)) {
						m = substr(s, RSTART, RLENGTH)
						if (m !~ ok) { print $1; break }
						s = substr(s, RSTART + RLENGTH)
					}
				}' | sort -u)
		else
			orte=$(printf '%s\n' "$inhalt" | grep -v "^$selbst	" | grep -E -- "$re" | cut -f1 | sort -u)
		fi
		for o in $orte; do
			melde "$art in $o"
		done
	done
	IFS=$alt_ifs
}

# pruefe_pfade: meldet gesperrte Dateien aus $dateien.
pruefe_pfade() {
	treffer=$(printf '%s\n' "$dateien" | grep -E "$pfade")
	alt_ifs=$IFS
	IFS='
'
	for d in $treffer; do
		melde "gesperrte Datei: $d"
	done
	IFS=$alt_ifs
}

# pruefe_adresse ORT ADRESSE: Autor/Committer muss eine GitHub-noreply-Adresse sein.
pruefe_adresse() {
	if ! printf '%s\n' "$2" | grep -qE "$autor_erlaubt"; then
		melde "$1 ist keine GitHub-noreply-Adresse (git config user.email <id>+<name>@users.noreply.github.com)"
	fi
}

case "$modus" in
--staged)
	dateien=$(git diff --cached --name-only --diff-filter=ACMR)
	# hinzugefügte Zeilen als "datei<TAB>zeile"
	inhalt=$(git diff --cached -U0 --no-color --diff-filter=ACMR | awk '
		/^\+\+\+ / { f = substr($0, 7); next }
		/^\+/ { print f "\t" substr($0, 2) }')
	pruefe_pfade
	pruefe_inhalt
	pruefe_adresse "Autor-Adresse" "$(git var GIT_AUTHOR_IDENT | sed -E 's/.*<(.*)>.*/\1/')"
	pruefe_adresse "Committer-Adresse" "$(git var GIT_COMMITTER_IDENT | sed -E 's/.*<(.*)>.*/\1/')"
	;;
--nachricht)
	datei="${2:?Aufruf: $0 --nachricht DATEI}"
	# Kommentarzeilen der Vorlage auslassen
	inhalt=$(grep -v '^#' "$datei" | sed 's/^/Commit-Nachricht\t/')
	pruefe_inhalt
	;;
--alle)
	dateien=$(git ls-files)
	# alle Zeilen als "datei<TAB>zeile" (Binärdateien ausgenommen)
	inhalt=$(git ls-files -z | xargs -0 grep -I -H '' 2>/dev/null | sed 's/:/\t/')
	pruefe_pfade
	pruefe_inhalt
	# alle Commits: Adressen und Nachrichten
	for c in $(git rev-list --all); do
		k=$(git rev-parse --short "$c")
		pruefe_adresse "Commit $k: Autor-Adresse" "$(git log -1 --format=%ae "$c")"
		pruefe_adresse "Commit $k: Committer-Adresse" "$(git log -1 --format=%ce "$c")"
		inhalt=$(git log -1 --format=%B "$c" | sed "s/^/Commit $k: Nachricht\t/")
		pruefe_inhalt
	done
	;;
*)
	echo "Aufruf: $0 --staged | --nachricht DATEI | --alle" >&2
	exit 2
	;;
esac

if [ "$fund" -ne 0 ]; then
	echo "datenschutz: Abbruch. Erlaubt sind nur Beispieladressen (@example.com, *.invalid) und GitHub-noreply-Adressen." >&2
	exit 1
fi
exit 0
