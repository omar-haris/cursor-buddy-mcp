# Internationalization Guidelines
- category: development
- priority: recommended

## Overview
Guidelines for implementing internationalization (i18n) in the project.

## Text Handling
- Use string keys instead of hardcoded text
- Implement proper pluralization rules
- Support RTL (Right-to-Left) languages
- Use Unicode (UTF-8) encoding everywhere

## Date and Time
- Use locale-aware date/time formatting
- Store dates in UTC, display in user's timezone
- Handle different calendar systems
- Support various date formats (MM/DD/YYYY vs DD/MM/YYYY)

## Numbers and Currency
- Use locale-specific number formatting
- Support different decimal separators (, vs .)
- Handle currency symbols and positioning
- Implement proper number grouping

## API Design
- Accept `Accept-Language` header
- Return localized error messages
- Support language fallbacks
- Use language codes (en-US, fr-FR, etc.)

## Testing
- Test with pseudo-localization
- Verify text expansion/contraction
- Test with different character sets
- Validate RTL layout behavior

## Performance
- Lazy load translation files
- Cache translations appropriately
- Minimize bundle size per locale
- Use translation keys efficiently 