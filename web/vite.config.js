import {
  defineConfig
} from 'vite'
import vue from '@vitejs/plugin-vue'
import tailwindcss from '@tailwindcss/vite'
// https://vite.dev/config/
export default defineConfig({
  plugins: [
    vue(),
    tailwindcss(),
  ],
  resolve: {
    alias: {
      events: 'events/',
    },
  },
  server: {
    host: '0.0.0.0',
    proxy: {
      '/bc': {
        target: 'https://fuhncvpxdcuf.ap-northeast-1.clawcloudrun.com',
        changeOrigin: true,
        rewrite: (path) => path.replace(/^\/bc/, ''),
      },
    },
  }
})


