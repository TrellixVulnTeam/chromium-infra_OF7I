import  { sassPlugin } from 'esbuild-sass-plugin';
import esbuild from 'esbuild';

esbuild.build({
    entryPoints: ['src/index.ts'],
    bundle: true,
    outfile: 'dist/main.js',
    minify: true,
    sourcemap: true,
    plugins: [sassPlugin()],
    watch: true,
}).catch((e) => {
    console.log(e);
})