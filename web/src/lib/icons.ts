// The only place icons are imported. Icons are Lucide icons, named in kebab-case
// as in cashflow_categories.csv. The names are those of the categories in the
// mobile cashflow app; add a name here before a category uses it.
import type { Component } from 'svelte';
import ArrowLeftRight from '@lucide/svelte/icons/arrow-left-right';
import Bell from '@lucide/svelte/icons/bell';
import Award from '@lucide/svelte/icons/award';
import ChartLine from '@lucide/svelte/icons/chart-line';
import Briefcase from '@lucide/svelte/icons/briefcase';
import CarFront from '@lucide/svelte/icons/car-front';
import CircleQuestionMark from '@lucide/svelte/icons/circle-question-mark';
import Droplets from '@lucide/svelte/icons/droplets';
import Ellipsis from '@lucide/svelte/icons/ellipsis';
import Gamepad2 from '@lucide/svelte/icons/gamepad-2';
import Gift from '@lucide/svelte/icons/gift';
import GraduationCap from '@lucide/svelte/icons/graduation-cap';
import HeartPulse from '@lucide/svelte/icons/heart-pulse';
import House from '@lucide/svelte/icons/house';
import Landmark from '@lucide/svelte/icons/landmark';
import LogOut from '@lucide/svelte/icons/log-out';
import Menu from '@lucide/svelte/icons/menu';
import PanelLeftClose from '@lucide/svelte/icons/panel-left-close';
import PanelLeftOpen from '@lucide/svelte/icons/panel-left-open';
import Percent from '@lucide/svelte/icons/percent';
import PiggyBank from '@lucide/svelte/icons/piggy-bank';
import Shapes from '@lucide/svelte/icons/shapes';
import Shield from '@lucide/svelte/icons/shield';
import ShieldCheck from '@lucide/svelte/icons/shield-check';
import ShoppingBasket from '@lucide/svelte/icons/shopping-basket';
import Smartphone from '@lucide/svelte/icons/smartphone';
import Target from '@lucide/svelte/icons/target';
import TrendingUp from '@lucide/svelte/icons/trending-up';
import Settings from '@lucide/svelte/icons/settings';
import Vault from '@lucide/svelte/icons/vault';
import Utensils from '@lucide/svelte/icons/utensils';
import Zap from '@lucide/svelte/icons/zap';

// Every Lucide icon accepts the same props (size, color, strokeWidth, ...).
type IconComponent = Component<any>;

const registry = {
  'arrow-left-right': ArrowLeftRight,
  award: Award,
  bell: Bell,
  briefcase: Briefcase,
  'chart-line': ChartLine,
  'car-front': CarFront,
  'circle-question-mark': CircleQuestionMark,
  droplets: Droplets,
  ellipsis: Ellipsis,
  'gamepad-2': Gamepad2,
  gift: Gift,
  'graduation-cap': GraduationCap,
  'heart-pulse': HeartPulse,
  house: House,
  landmark: Landmark,
  'log-out': LogOut,
  menu: Menu,
  'panel-left-close': PanelLeftClose,
  'panel-left-open': PanelLeftOpen,
  percent: Percent,
  'piggy-bank': PiggyBank,
  shapes: Shapes,
  shield: Shield,
  'shield-check': ShieldCheck,
  'shopping-basket': ShoppingBasket,
  smartphone: Smartphone,
  target: Target,
  'trending-up': TrendingUp,
  utensils: Utensils,
  settings: Settings,
  vault: Vault,
  zap: Zap,
} satisfies Record<string, IconComponent>;

/** The Lucide names that can be drawn. */
export type IconName = keyof typeof registry;

/** Every registered icon name. */
export const ICON_NAMES = Object.keys(registry) as IconName[];

/** What is drawn for a name that is not registered (the file names an icon this app lacks). */
export const UNKNOWN_ICON: IconName = 'circle-question-mark';

/** The component for an icon name; an unregistered name gets the question mark. */
export function iconComponent(name: string): IconComponent {
  return Object.hasOwn(registry, name) ? registry[name as IconName] : registry[UNKNOWN_ICON];
}
