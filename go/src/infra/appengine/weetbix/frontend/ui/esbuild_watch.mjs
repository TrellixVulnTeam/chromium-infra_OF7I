import  { sassPlugin } from 'esbuild-sass-plugin';
import esbuild from 'esbuild';

esbuild.build({
    entryPoints: ['index.tsx'],
    bundle: true,
    outfile: 'dist/main.js',
    sourcemap: true,
    logLevel: 'debug',
    plugins: [sassPlugin()],
    watch: true,
}).catch((e) => {
    console.log(e);
})