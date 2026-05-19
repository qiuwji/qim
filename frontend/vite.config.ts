import { defineConfig } from 'vitest/config';
import react from '@vitejs/plugin-react';
import tailwindcss from '@tailwindcss/vite';
import basicSsl from '@vitejs/plugin-basic-ssl';
import path from 'path';

export default defineConfig({
  resolve: {
    alias: { '@': path.resolve(__dirname, 'src') }
  },
  plugins: [react(), tailwindcss(), basicSsl()],
  test: {
    coverage: {
      provider: 'v8',
      reporter: ['text', 'json-summary'],
      include: [
        'src/hooks/chat/models/contactViewModel.ts',
        'src/hooks/chat/reducers/chatReducer.ts',
        'src/hooks/chat/models/conversationModel.ts',
        'src/hooks/chat/models/conversationViewModel.ts',
        'src/hooks/chat/models/friendModel.ts',
        'src/hooks/chat/models/messageModel.ts',
        'src/hooks/chat/models/realtimeModel.ts',
      ],
      exclude: ['src/hooks/chat/**/*.test.{ts,tsx}'],
      thresholds: {
        statements: 80,
        branches: 70,
        functions: 80,
        lines: 80,
      },
    },
  },
  server: {
    host: '0.0.0.0',
    port: 5173,
    proxy: {
      '/api': 'http://localhost:8080',
      '/uploads': 'http://localhost:8080',
      '/ws': {
        target: 'ws://localhost:8080',
        ws: true
      }
    }
  }
});
