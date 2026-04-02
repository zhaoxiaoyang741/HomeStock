import type { ElementType, ComponentPropsWithoutRef, ReactNode } from 'react'
import { cva, type VariantProps } from 'class-variance-authority'
import { cn } from '@/lib/utils'

const boxVariants = cva('w-full', {
  variants: {
    variant: {
      default: 'bg-surface text-on-surface',
      low:     'bg-surface-container-low text-on-surface',
      high:    'bg-surface-container-high text-on-surface',
      inset:   'bg-surface-container text-on-surface',
      primary: 'bg-primary-container text-on-primary-container',
    },
    rounded: {
      none: '',
      sm:   'rounded-sm',
      md:   'rounded',
      lg:   'rounded-lg',
    },
    bordered: {
      true:  'border border-outline-variant',
      false: '',
    },
  },
  defaultVariants: {
    variant: 'default',
    rounded: 'none',
    bordered: false,
  },
})

type BoxOwnProps<E extends ElementType> = {
  as?: E
  className?: string
  children?: ReactNode
} & VariantProps<typeof boxVariants>

type BoxProps<E extends ElementType> = BoxOwnProps<E> &
  Omit<ComponentPropsWithoutRef<E>, keyof BoxOwnProps<E>>

export function Box<E extends ElementType = 'div'>({
  as,
  variant,
  rounded,
  bordered,
  className,
  children,
  ...props
}: BoxProps<E>) {
  const Comp = as ?? 'div'
  return (
    <Comp
      className={cn(boxVariants({ variant, rounded, bordered }), className)}
      {...props}
    >
      {children}
    </Comp>
  )
}
