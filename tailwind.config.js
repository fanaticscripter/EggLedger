const colors = require('tailwindcss/colors');

module.exports = {
  content: ['./www/*.html'],
  theme: {
    extend: {
      colors: {
        green: colors.emerald,
      },
      height: {
        stretch: 'stretch',
      },
    },
  },
  plugins: [require('@tailwindcss/forms')],
};
