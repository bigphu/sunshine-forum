// Sinh kiểu TypeScript từ openapi.yaml của từng module backend vào src/lib/api/<module>.d.ts.
// Chạy lại sau khi sửa spec (npm run gen:api) và commit file sinh ra; không sửa tay.
import { execFileSync } from 'node:child_process'

const modules = ['identity', 'screening', 'content', 'moderation', 'penalties', 'notifications', 'administration']
for (const m of modules) {
  execFileSync(
    process.execPath,
    [
      'node_modules/openapi-typescript/bin/cli.js',
      `../backend/internal/${m}/handler/openapi.yaml`,
      '-o',
      `src/lib/api/${m}.d.ts`,
    ],
    { stdio: 'inherit' },
  )
}
