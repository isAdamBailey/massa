module.exports = {
  apps: [
    {
      name: 'massa-web',
      script: '.output/server/index.mjs',
      cwd: __dirname,
      instances: 1,
      exec_mode: 'fork',
      env: {
        NODE_ENV: 'production',
        HOST: '127.0.0.1',
      },
    },
  ],
}
