# Hero Section Research: Additional Developer-Focused Sites

Research conducted on February 5, 2026, analyzing hero section design patterns from three leading developer-focused platforms.

---

## 1. Stripe (stripe.com)

### Layout Structure
- **Single-column centered layout** with vertical stacking
- Primary heading anchors the top, followed by descriptive body text
- Call-to-action buttons stack vertically below the text for mobile responsiveness
- Maximum width constraints maintain readability on large screens

### Visual Elements
- **Decorative wave illustration** positioned beneath CTA buttons adds visual interest without overwhelming content
- Minimal design approach keeps focus on messaging
- Clean, sophisticated aesthetic aligned with financial services branding
- Client-variant wrapper elements suggest conditional rendering for personalization

### Code/Product Previews
- No traditional code snippets in the hero section
- Product value communicated through messaging rather than technical demonstrations
- Visual focus on brand partnerships over feature showcases

### Social Proof Elements
- **Carousel of customer logos** (Amazon, Shopify, N26, Google, Figma)
- "Visual credibility through association" approach
- Emphasizes enterprise adoption and scale
- Rotates through recognizable brands to communicate trustworthiness

### Typography Hierarchy
- **Large sans-serif heading** dominates the visual hierarchy
- Smaller supporting text provides context
- Clear hierarchy: Headline > Subtext > CTAs
- Value proposition stated directly: "Financial infrastructure for revenue growth"

### Background Treatments
- Minimal decorative elements maintain content focus
- Single wave illustration adds subtle sophistication
- Clean backgrounds emphasize text legibility

### Interactive Elements
- Two primary CTAs: "Start now" and "Sign in with Google"
- Logo carousel invites passive engagement
- Social authentication pathway reduces friction

---

## 2. Linear (linear.app)

### Layout Structure
- **Centered, single-column design** with maximum width constraints
- Generous vertical spacing creates breathing room between elements
- Content stacks vertically with balanced proportions
- Responsive design adapts from multi-line desktop to single-column mobile

### Visual Elements
- **Dark theme styling** as the default presentation
- CSS custom properties control color schemes throughout
- Gradient treatments on text: `linear-gradient(to right, var(--color-text-primary), transparent 80%)`
- Dynamic theme switching between light/dark with system preference detection
- Overlapping avatar arrangements in icon piles

### Code/Product Previews
- **UI mockups** showcasing product functionality
- Project management boards with status indicators
- Issue tracking examples (bug reports, feature requests)
- Timeline and progress visualizations with percentage completion
- Team member avatars and activity feeds

### Social Proof Elements
- "Powering the world's best product teams" positioning statement
- Customer testimonials and case studies section
- Company logos in dedicated section
- Icon piles with overlapping avatars suggest active teams

### Typography Hierarchy
Multi-tier text system with CSS custom properties:
- **Title-8 size** for primary headlines (medium font-weight)
- **Title-2 and Title-3** for section headers
- **Text-regular and text-small** for body copy in tertiary colors
- **Title-1** for prominent display text
- `text-wrap: balance` for improved heading readability

### Background Treatments
- **Dark backgrounds** (#08090a theme color)
- Tertiary-colored text creates contrast layers
- CSS custom properties (`--height` variables) ensure consistent spacing rhythm
- Dark mode creates premium, focused aesthetic

### Interactive Elements
- Navigation links in header and footer
- Call-to-action buttons: "Start building," "Learn more"
- Announcement banner linking to beta features
- Responsive menu structures with skip navigation for accessibility

---

## 3. Supabase (supabase.com)

### Layout Structure
- **Centered, single-column layout** with vertical stacking
- Main heading aligns center, followed by descriptive text
- Dual call-to-action buttons positioned below
- Clean separation between hero content and subsequent sections

### Visual Elements
- **Theme toggle** handling dark/light mode transitions
- Animated color scheme switching on mode change
- Logo imagery displays both light and dark variants
- Seamless visual adaptation between themes

### Code/Product Previews
- **CLI command preview** in Edge Functions section: `$ supabase functions deploy`
- Provides immediate technical context for developers
- Demonstrates product functionality through actual commands
- Developer-first approach with code-centric previews

### Social Proof Elements
- "Trusted by fast-growing companies worldwide" positioning
- **Carousel of 16+ company logos**: Mozilla, GitHub, 1Password, and others
- Continuous scrolling animation effect
- Repeated carousel structure suggests infinite scroll pattern

### Typography Hierarchy
- **Primary headline**: Large, bold sans-serif emphasizing dual value proposition
- Headline: "Build in a weekend. Scale to millions."
- **Subheading**: Medium-weight body text explaining core features
- **CTAs**: Button-styled text with contrasting colors for action emphasis

### Background Treatments
- **System preference detection** for dark/light mode
- HTML style attributes and data attributes manage visual presentation
- Adaptive backgrounds based on user preferences
- Clean, minimal background allows content focus

### Interactive Elements
- Two prominent buttons: "Start your project" (primary) and "Request a demo" (secondary)
- Theme toggle for user preference
- Logo carousel with continuous animation
- Multiple engagement paths for different user intents

---

## Key Design Patterns Summary

### Common Patterns Across All Three Sites

| Pattern | Stripe | Linear | Supabase |
|---------|--------|--------|----------|
| Centered single-column layout | Yes | Yes | Yes |
| Dark mode support | Partial | Primary | Full toggle |
| Logo carousel for social proof | Yes | Yes | Yes |
| Dual CTA buttons | Yes | Yes | Yes |
| Minimal background treatments | Yes | Yes | Yes |
| Code/product previews | No | UI mockups | CLI commands |

### Differentiated Approaches

1. **Stripe**: Enterprise credibility through brand associations; minimal technical demonstration; financial services aesthetic
2. **Linear**: Dark-first design; product-focused UI mockups; sophisticated gradient effects; developer tool aesthetic
3. **Supabase**: Developer-first with CLI previews; adaptive theming; scale-focused messaging; startup/growth aesthetic

### Actionable Insights for Wave Hero Section

1. **Layout**: Adopt centered single-column layout with maximum width constraints
2. **Theme**: Consider dark mode as default or primary option (Linear pattern)
3. **Social Proof**: Implement logo carousel if customer logos are available
4. **Code Previews**: Include CLI command examples (Supabase pattern) showing Wave commands
5. **Typography**: Use clear hierarchy with large headline, supporting text, and prominent CTAs
6. **CTAs**: Dual button pattern - primary action + secondary engagement
7. **Background**: Keep minimal, use subtle gradients or illustrations (Stripe wave pattern)
8. **Interactivity**: Theme toggle, smooth animations on mode transitions
