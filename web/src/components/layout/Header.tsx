import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { Moon, Sun, PanelLeftClose, PanelLeftOpen, Search, Languages } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar'
import { Input } from '@/components/ui/input'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { cn } from '@/lib/utils'
import { useAppStore } from '@/store/appStore'
import { useAuthStore } from '@/store/authStore'
import { useTheme } from '@/hooks/useTheme'

export default function Header() {
  const { collapsed, toggleCollapsed } = useAppStore()
  const { resolvedTheme, setTheme } = useTheme()
  const { t, i18n } = useTranslation('header')
  const user = useAuthStore((s) => s.user)

  const [searchQuery, setSearchQuery] = useState('')
  const navigate = useNavigate()

  function handleSearchKeyDown(event: React.KeyboardEvent<HTMLInputElement>) {
    if (event.key === 'Enter' && searchQuery.trim()) {
      navigate(`/inventory?keyword=${encodeURIComponent(searchQuery.trim())}`)
    }
  }

  function handleLanguageChange(lang: string) {
    void i18n.changeLanguage(lang)
    try { localStorage.setItem('language', lang) } catch {}
    document.documentElement.lang = lang
  }

  return (
    <header className={cn(
      'fixed top-0 right-0 h-16 flex items-center gap-4 px-4 bg-surface/80 glass-effect border-b border-outline-variant z-10',
      'transition-[left] duration-300 ease-in-out',
      collapsed ? 'left-16' : 'left-50'
    )}>
      {/* Sidebar toggle */}
      <button
        onClick={toggleCollapsed}
        className="p-2 rounded-lg cursor-pointer text-on-surface-variant hover:bg-surface-container hover:text-on-surface transition-colors shrink-0"
        aria-label={collapsed ? t('toggleSidebarOpen') : t('toggleSidebarClose')}
      >
        {collapsed
          ? <PanelLeftOpen className="w-5 h-5" />
          : <PanelLeftClose className="w-5 h-5" />
        }
      </button>

      {/* Search */}
      <div className="relative w-72">
        <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-on-surface-variant" />
        <Input
          className="pl-9 bg-surface-container-low border-outline-variant"
          placeholder={t('searchPlaceholder')}
          value={searchQuery}
          onChange={(e) => setSearchQuery(e.target.value)}
          onKeyDown={handleSearchKeyDown}
        />
      </div>

      {/* Actions */}
      <div className="flex items-center gap-2 ml-auto">
        {/* Language switcher */}
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <button
              className="p-2 rounded-lg text-on-surface-variant hover:bg-surface-container transition-colors"
              aria-label={t('switchLanguage')}
            >
              <Languages className="w-5 h-5 cursor-pointer" />
            </button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end" className="w-36">
            <DropdownMenuRadioGroup value={i18n.language} onValueChange={handleLanguageChange}>
              <DropdownMenuRadioItem value="zh-CN">中文</DropdownMenuRadioItem>
              <DropdownMenuRadioItem value="en">English</DropdownMenuRadioItem>
            </DropdownMenuRadioGroup>
          </DropdownMenuContent>
        </DropdownMenu>

        <button
          className="p-2 rounded-lg text-on-surface-variant hover:bg-surface-container transition-colors"
          aria-label={resolvedTheme === 'dark' ? t('toggleLight') : t('toggleDark')}
          onClick={() => setTheme(resolvedTheme === 'dark' ? 'light' : 'dark')}
        >
          {resolvedTheme === 'dark'
            ? <Sun className="w-5 h-5 cursor-pointer" />
            : <Moon className="w-5 h-5 cursor-pointer" />
          }
        </button>
        <Avatar className="w-8 h-8 ml-1 cursor-pointer">
          <AvatarImage src="" alt={t('userAvatar')} />
          <AvatarFallback className="bg-primary text-on-primary text-xs">
            {user?.display_name?.charAt(0) ?? t('userFallback')}
          </AvatarFallback>
        </Avatar>
      </div>
    </header>
  )
}
