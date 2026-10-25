import * as path from 'path'
import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

const BASE_URL = './'

export default defineConfig({
  plugins: [vue()],
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },
  optimizeDeps: {
    include: [
      'ts-md5/dist/md5',
    ],
    exclude: [
      'ts-md5',
    ],
  },
  assetsDir: '.',
  base: BASE_URL,
  port: 8000,
  server: {
    proxy: {
      '/api': 'http://localhost:8080',
    },
  },
  build: {
    outDir: 'dist',
    emptyOutDir: true,
  },
  configureServer: ({ root, app, watcher }) => {
    watcher.add(path.resolve(root, './public/**/*'))
    const publicPath = path.resolve(root, './public')
    watcher.on('change', function (path) {
      console.log(path)
      if (path.startsWith(publicPath)) {
        watcher.send({
          type: 'full-reload',
          path,
        })
      }
    })
  },
})
