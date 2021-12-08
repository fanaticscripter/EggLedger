module.exports = {
  mode: 'jit',
  purge: ['./www/*.html'],
  darkMode: false,
  theme: {
    extend: {
      height: {
        stretch: 'stretch',
      },
    },
  },
  plugins: [require('@tailwindcss/forms')],
};
