import { defineConfig } from 'vite';
import { svelte } from '@sveltejs/vite-plugin-svelte';

export default defineConfig({
	plugins: [svelte()],
	server: {
		host: '0.0.0.0',
		port: 5173,
		hmr: {
			clientPort: Number(process.env.VITE_HMR_CLIENT_PORT ?? 0) || undefined
		},
		watch: {
			usePolling: process.env.VITE_USE_POLLING === 'true'
		},
		proxy: {
			'/api': {
				target: process.env.VITE_API_PROXY_TARGET ?? 'http://localhost:8080',
				changeOrigin: true
			}
		}
	}
});
