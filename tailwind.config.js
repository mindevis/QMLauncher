/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./index.html",
    "./src/renderer/**/*.{js,ts,jsx,tsx}",
  ],
  theme: {
    extend: {
      colors: {
        // Minecraft themed colors
        minecraft: {
          grass: "#7CB342",
          "grass-dark": "#558B2F",
          dirt: "#8D6E63",
          stone: "#757575",
          wood: "#8D6E63",
          water: "#2196F3",
          lava: "#FF5722",
          "button-green": "#7CB342",
          "button-green-dark": "#558B2F",
          "button-green-darker": "#33691E",
        },
      },
      fontFamily: {
        minecraft: ["Minecraft", "monospace"],
      },
    },
  },
  plugins: [],
}
