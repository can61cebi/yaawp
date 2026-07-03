#!/usr/bin/env bash
#
# install.sh builds yaawp and installs it into a user prefix (default
# ~/.local), so it shows up in the KDE application launcher, delivers
# notifications, and can be set to start on login from its settings.
#
# Usage:
#   ./install.sh              build and install into ~/.local
#   ./install.sh --uninstall  remove the installed files
#   PREFIX=/some/path ./install.sh   install into a different prefix
#
set -euo pipefail

PREFIX="${PREFIX:-$HOME/.local}"
ROOT="$(cd "$(dirname "$0")" && pwd)"

APP_ID="tr.cebi.yaawp"
INSTALLED=(
    "$PREFIX/bin/yaawp"
    "$PREFIX/bin/yaawp-daemon"
    "$PREFIX/share/applications/$APP_ID.desktop"
    "$PREFIX/share/knotifications6/yaawp.notifyrc"
    "$PREFIX/share/icons/hicolor/scalable/apps/$APP_ID.svg"
)

msg()  { printf '\033[1;32m==>\033[0m %s\n' "$*"; }
warn() { printf '\033[1;33m==>\033[0m %s\n' "$*" >&2; }
die()  { printf '\033[1;31m==>\033[0m %s\n' "$*" >&2; exit 1; }

refresh_caches() {
    command -v update-desktop-database >/dev/null 2>&1 && \
        update-desktop-database "$PREFIX/share/applications" >/dev/null 2>&1 || true
    command -v gtk-update-icon-cache >/dev/null 2>&1 && \
        gtk-update-icon-cache -q -t -f "$PREFIX/share/icons/hicolor" >/dev/null 2>&1 || true
    command -v kbuildsycoca6 >/dev/null 2>&1 && kbuildsycoca6 >/dev/null 2>&1 || true
}

uninstall() {
    msg "Removing yaawp from $PREFIX"
    for f in "${INSTALLED[@]}"; do
        [ -e "$f" ] && rm -f "$f" && echo "  removed $f"
    done
    rm -f "$HOME/.config/autostart/$APP_ID.desktop"
    refresh_caches
    msg "Done. Your session data in ~/.local/share/yaawp was left untouched."
    echo "     Remove it by hand if you want a clean slate:"
    echo "       rm -rf ~/.local/share/yaawp ~/.cache/yaawp"
}

install_app() {
    for tool in go cmake ninja; do
        command -v "$tool" >/dev/null 2>&1 || die "missing build tool: $tool"
    done

    msg "Building the daemon"
    ( cd "$ROOT/daemon" && CGO_ENABLED=1 go build -trimpath -o bin/yaawp-daemon ./cmd/yaawp-daemon )

    msg "Building the GUI"
    local builddir="$ROOT/gui/build"
    # KDE install paths (share/applications, icons, ...) are baked in at configure
    # time from the prefix. If a previous configure used a different prefix, wipe
    # the cache so they are recomputed and land under this prefix.
    if [ -f "$builddir/CMakeCache.txt" ] && \
       ! grep -qx "CMAKE_INSTALL_PREFIX:PATH=$PREFIX" "$builddir/CMakeCache.txt"; then
        rm -rf "$builddir"
    fi
    cmake -S "$ROOT/gui" -B "$builddir" -G Ninja \
        -DCMAKE_INSTALL_PREFIX="$PREFIX" \
        -DCMAKE_BUILD_TYPE=Release
    cmake --build "$builddir"

    msg "Installing into $PREFIX"
    install -Dm755 "$ROOT/daemon/bin/yaawp-daemon" "$PREFIX/bin/yaawp-daemon"
    DESTDIR="" cmake --install "$ROOT/gui/build" >/dev/null

    # The launcher entry ships with a bare "Exec=yaawp" and a themed "Icon" name,
    # which assume the prefix bin dir is on PATH and the running shell has already
    # picked up the installed icon. For a user prefix neither is guaranteed, so
    # rewrite both to absolute paths. plasmashell then loads the icon file
    # directly and the entry shows correctly right after install.
    local desktop="$PREFIX/share/applications/$APP_ID.desktop"
    local iconfile="$PREFIX/share/icons/hicolor/scalable/apps/$APP_ID.svg"
    if [ -f "$desktop" ]; then
        sed -i "s|^Exec=yaawp$|Exec=$PREFIX/bin/yaawp|" "$desktop"
        sed -i "s|^Icon=$APP_ID$|Icon=$iconfile|" "$desktop"
    fi

    refresh_caches

    msg "Installed."
    case ":$PATH:" in
        *":$PREFIX/bin:"*) ;;
        *) warn "$PREFIX/bin is not in your PATH. Add it so 'yaawp' resolves:"
           echo "       export PATH=\"$PREFIX/bin:\$PATH\"" ;;
    esac
    echo
    echo "  Launch it from the application menu (search for yaawp) or run: yaawp"
    echo "  To open it automatically on login, enable 'Start on login' in Settings."
}

if [ "${1:-}" = "--uninstall" ] || [ "${1:-}" = "-u" ]; then
    uninstall
else
    install_app
fi
