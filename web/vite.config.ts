import { readdirSync } from 'node:fs';
import path from 'node:path';
import process from 'node:process';
import VueI18nPlugin from '@intlify/unplugin-vue-i18n/vite';
import tailwindcss from '@tailwindcss/vite';
import vue from '@vitejs/plugin-vue';
import dotenv from 'dotenv';
import type { Plugin, ProxyOptions } from 'vite';
import prismjs from 'vite-plugin-prismjs';
import svgLoader from 'vite-svg-loader';
import type { ViteUserConfig } from 'vitest/config';
import { defineConfig } from 'vitest/config';

dotenv.config({ path: path.resolve(__dirname, '../.env'), quiet: true });

const getEnvString = (envVar: string | undefined) => (envVar != null && envVar !== '' ? envVar : undefined);
const viteUserSessCookie = getEnvString(process.env.VITE_DEV_USER_SESS_COOKIE);
const viteDevProxy = getEnvString(process.env.VITE_DEV_PROXY);

function createProxyOptions(): ProxyOptions {
  const options: ProxyOptions = {
    target: viteDevProxy,
    changeOrigin: true,
  };

  if (viteUserSessCookie !== undefined) {
    options.configure = (proxy) => {
      proxy.on('proxyReq', (proxyReq, req) => {
        const existingHeader = proxyReq.getHeader('cookie');
        const headerValues: string[] = [];

        if (Array.isArray(existingHeader)) {
          headerValues.push(...existingHeader);
        } else if (typeof existingHeader === 'string' && existingHeader.length > 0) {
          headerValues.push(existingHeader);
        }

        const hasSessionCookie = headerValues.some((value) => value.includes('user_sess='));

        if (!hasSessionCookie) {
          headerValues.push(`user_sess=${viteUserSessCookie}`);
          proxyReq.setHeader('cookie', headerValues.join('; '));
        }
      });
    };
  }

  return options;
}

function woodpeckerInfoPlugin(): Plugin {
  return {
    name: 'woodpecker-info',
    configureServer() {
      if (viteDevProxy !== undefined) {
        console.log(
          [
            `Using dev server with proxy to existing Woodpecker server running at: ${viteDevProxy}`,
            '\n  🚀 Access the UI at http://localhost:8010/',
          ].join('\n'),
        );
        return;
      }

      console.log(
        [
          '1) Please add `WOODPECKER_DEV_WWW_PROXY=http://localhost:8010` to your `.env` file.',
          '2) Start the Woodpecker server',
          '3) If you want to run the vite dev server (`pnpm start`) within a container please set `VITE_DEV_SERVER_HOST=0.0.0.0`.',
          `\n  🚀 Access the UI at http://localhost:8000/`,
        ].join('\n'),
      );
    },
  };
}

function externalCSSPlugin(): Plugin {
  return {
    name: 'external-css',
    transformIndexHtml: {
      order: 'post',
      handler() {
        return [
          {
            tag: 'link',
            attrs: { rel: 'stylesheet', type: 'text/css', href: '/assets/custom.css' },
            injectTo: 'head',
          },
        ];
      },
    },
  };
}

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [
    vue(),
    VueI18nPlugin({
      include: path.resolve(__dirname, 'src/assets/locales/**'),
    }),
    (() => {
      const virtualModuleId = 'virtual:vue-i18n-supported-locales';
      const resolvedVirtualModuleId = `\0${virtualModuleId}`;

      const filenames = readdirSync('src/assets/locales/').map((filename) => filename.replace('.json', ''));

      return {
        name: 'vue-i18n-supported-locales',

        resolveId(id) {
          if (id === virtualModuleId) {
            return resolvedVirtualModuleId;
          }
        },

        load(id) {
          if (id === resolvedVirtualModuleId) {
            return `export const SUPPORTED_LOCALES = ${JSON.stringify(filenames)}`;
          }
        },
      };
    })(),
    svgLoader(),
    externalCSSPlugin(),
    woodpeckerInfoPlugin(),
    prismjs({
      languages: ['yaml'],
    }),
    tailwindcss(),
  ],
  resolve: {
    alias: {
      '~/': `${path.resolve(__dirname, 'src')}/`,
    },
  },
  logLevel: 'warn',
  server: {
    allowedHosts: true,
    host: process.env.VITE_DEV_SERVER_HOST ?? '127.0.0.1',
    port: 8010,
    proxy:
      viteDevProxy !== undefined
        ? {
            '/api': createProxyOptions(),
            '/web-config.js': createProxyOptions(),
            '/authorize': {
              target: viteDevProxy,
              changeOrigin: true,
            },
          }
        : undefined,
  },
  test: {
    globals: true,
    environment: 'jsdom',
  },
} as ViteUserConfig);
