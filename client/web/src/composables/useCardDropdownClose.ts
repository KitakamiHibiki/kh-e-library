type CloseFn = () => void

// All card dropdowns registered for right-click closing. Each card registers its
// "阅读状态" and "更多" dropdowns; BookShelf registers the shared context menu.
const closers = new Set<CloseFn>()

let installed = false

// Right-button clicks never fire a "click" event, so el-dropdown's built-in
// outside-click handling (backed by @vueuse onClickOutside, which closes on
// "click") never closes a menu when the user right-clicks elsewhere. Left-clicks
// already work natively; this closes the gap for right-clicks by closing every
// registered card dropdown on a right-button pointerdown (which fires before the
// contextmenu event, so a context menu about to open starts from a clean slate).
function install() {
  if (installed) return
  installed = true
  document.addEventListener(
    'pointerdown',
    (e: PointerEvent) => {
      if (e.button !== 2) return
      closers.forEach((fn) => fn())
    },
    true,
  )
}

export function useCardDropdownClose() {
  const register = (fn: CloseFn): (() => void) => {
    install()
    closers.add(fn)
    return () => {
      closers.delete(fn)
    }
  }

  return { register }
}
