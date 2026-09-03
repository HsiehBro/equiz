---
name: Academic Focus
colors:
  surface: '#f8f9fa'
  surface-dim: '#d9dadb'
  surface-bright: '#f8f9fa'
  surface-container-lowest: '#ffffff'
  surface-container-low: '#f3f4f5'
  surface-container: '#edeeef'
  surface-container-high: '#e7e8e9'
  surface-container-highest: '#e1e3e4'
  on-surface: '#191c1d'
  on-surface-variant: '#414755'
  inverse-surface: '#2e3132'
  inverse-on-surface: '#f0f1f2'
  outline: '#717786'
  outline-variant: '#c1c6d7'
  surface-tint: '#005bc1'
  primary: '#0058bc'
  on-primary: '#ffffff'
  primary-container: '#0070eb'
  on-primary-container: '#fefcff'
  inverse-primary: '#adc6ff'
  secondary: '#006e28'
  on-secondary: '#ffffff'
  secondary-container: '#6ffb85'
  on-secondary-container: '#00732a'
  tertiary: '#bc000a'
  on-tertiary: '#ffffff'
  tertiary-container: '#e2241f'
  on-tertiary-container: '#fffbff'
  error: '#ba1a1a'
  on-error: '#ffffff'
  error-container: '#ffdad6'
  on-error-container: '#93000a'
  primary-fixed: '#d8e2ff'
  primary-fixed-dim: '#adc6ff'
  on-primary-fixed: '#001a41'
  on-primary-fixed-variant: '#004493'
  secondary-fixed: '#72fe88'
  secondary-fixed-dim: '#53e16f'
  on-secondary-fixed: '#002107'
  on-secondary-fixed-variant: '#00531c'
  tertiary-fixed: '#ffdad5'
  tertiary-fixed-dim: '#ffb4aa'
  on-tertiary-fixed: '#410001'
  on-tertiary-fixed-variant: '#930005'
  background: '#f8f9fa'
  on-background: '#191c1d'
  surface-variant: '#e1e3e4'
typography:
  display:
    fontFamily: Inter
    fontSize: 30px
    fontWeight: '700'
    lineHeight: 38px
    letterSpacing: -0.5px
  headline-lg:
    fontFamily: Inter
    fontSize: 22px
    fontWeight: '600'
    lineHeight: 28px
  headline-md:
    fontFamily: Inter
    fontSize: 18px
    fontWeight: '600'
    lineHeight: 24px
  body-lg:
    fontFamily: Inter
    fontSize: 17px
    fontWeight: '400'
    lineHeight: 26px
  body-md:
    fontFamily: Inter
    fontSize: 15px
    fontWeight: '400'
    lineHeight: 22px
  label-md:
    fontFamily: Inter
    fontSize: 13px
    fontWeight: '500'
    lineHeight: 18px
    letterSpacing: 0.2px
  label-sm:
    fontFamily: Inter
    fontSize: 11px
    fontWeight: '600'
    lineHeight: 16px
rounded:
  sm: 0.25rem
  DEFAULT: 0.5rem
  md: 0.75rem
  lg: 1rem
  xl: 1.5rem
  full: 9999px
spacing:
  base: 4px
  xs: 8px
  sm: 12px
  md: 16px
  lg: 24px
  xl: 32px
  margin-mobile: 16px
  gutter-card: 12px
---

## Brand & Style
The design system is centered on **Focus-Driven Minimalism**. For an educational exam prep environment, the UI acts as a silent facilitator, removing visual noise to maximize cognitive retention. The aesthetic is clean, professional, and trustworthy, utilizing ample whitespace to prevent "test anxiety" often associated with dense academic content.

The style leans into **Modern Corporate Minimalism** with subtle **Tonal Layering**. It prioritizes clarity and high legibility, ensuring that the interface feels like a premium utility rather than a distraction. The emotional response should be one of calm confidence and organized progress.

## Colors
The palette is functional and semantic. 
- **Primary (Study Blue):** Used for primary actions, active states, and branding elements. It evokes reliability.
- **Success (Green):** Specifically reserved for correct answers and completed progress states.
- **Error (Red):** Used for incorrect answers and critical alerts.
- **Neutral/Background:** A very light grey (#F8F9FA) distinguishes the background from white content cards (#FFFFFF), creating a soft hierarchy without harsh borders.

## Typography
The system uses **Inter**, a highly legible sans-serif optimized for mobile screens and small text. For WeChat Mini Programs, this ensures a native feel while maintaining a distinct professional edge.

- **Question Text:** Use `body-lg` with a slightly increased line height (1.6) to improve readability for long-form exam questions.
- **Option Text:** Use `body-md` for multiple-choice selections.
- **Hierarchy:** Use `headline-lg` for screen titles and `label-sm` for meta-information like "Question 5 of 20."

## Layout & Spacing
This design system utilizes a **Fixed Margin Fluid Grid**. The standard safe margin for all mobile screens is 16px. 

- **Vertical Rhythm:** A 4px baseline grid ensures consistent spacing between questions, options, and explanations. 
- **Card Spacing:** Question cards should be separated by 16px. Internal padding within cards is set to 20px to give content "room to breathe."
- **Stacking:** Elements within a question (text, image, options) should use 12px (sm) or 16px (md) gaps depending on content density.

## Elevation & Depth
Depth is achieved through **Tonal Separation** rather than heavy shadows to keep the interface feeling "light."

- **Level 0 (Background):** #F8F9FA.
- **Level 1 (Cards):** Pure white (#FFFFFF) with a very subtle 2px blur shadow (5% opacity) to provide a soft lift.
- **Level 2 (Active/Modal):** Use a slightly more pronounced shadow (10% opacity) for floating action buttons or pop-up explanations.
- **Dividers:** 1px hair-lines (#E5E5EA) are used only when necessary; preferred separation is achieved through whitespace.

## Shapes
The design system adopts a **Softly Rounded** geometric language. 
- **Primary Cards:** 12px (`rounded-lg`) corner radius for all question containers and dashboard widgets.
- **Buttons & Inputs:** 8px (`rounded-md`) for a professional yet accessible appearance.
- **Selection Circles:** Radio buttons for single-choice questions remain perfectly circular, while checkbox options for multi-select use a 4px radius.
- **Progress Bars:** Fully rounded ends (pill-shaped) to represent a "smooth" journey through the exam.

## Components
- **Question Cards:** Pure white containers with 12px corners. The question text sits at the top, followed by a vertical stack of interactive options.
- **Selection Options:** High-tap-area rows (min-height 48px). Use a subtle light blue fill (#EBF5FF) and a 1px Blue border for the "Selected" state.
- **Feedback States:** Upon submission, the selected option transitions to Green (Correct) or Red (Incorrect), accompanied by a "Solution Detail" card that slides up from the bottom.
- **Progress Header:** A slim, sticky bar at the top of the screen featuring a pill-shaped progress track and a digital timer.
- **Action Buttons:** Primary buttons use a solid #007AFF fill with white text. Secondary buttons (e.g., "Skip," "Save for later") use a "Ghost" style with a 1px blue outline.
- **Bottom Navigation:** Clean iconography with labels. The active icon uses the primary blue color, while inactive states use a neutral grey.