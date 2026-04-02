import { create } from 'zustand'

interface AppState {
  collapsed: boolean
  toggleCollapsed: () => void
  setCollapsed: (value: boolean) => void
}

export const useAppStore = create<AppState>((set) => ({
  collapsed: false,
  toggleCollapsed: () => set((state) => ({ collapsed: !state.collapsed })),
  setCollapsed: (value) => set({ collapsed: value }),
}))
