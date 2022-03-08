import  { sassPlugin } from 'esbuild-sass-plugin';
import esbuild from 'esbuild';

esbuild.build({
    entryPoints: ['index.tsx'],
    bundle: true,
    outfile: 'dist/main.js',
    minify: true,
    sourcemap: true,
    plugins: [sassPlugin()],
}).catch((e) => {
    process.exit(1);
})