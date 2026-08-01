import starlight from '@astrojs/starlight';
import { defineConfig } from 'astro/config';
import starlightThemeRapide from 'starlight-theme-rapide';

// GitHub Pages: served at https://michal-bartak.github.io/pg-accounts-management/
// (base matches the repo name). Update both if the repo is renamed.
export default defineConfig({
  site: 'https://michal-bartak.github.io',
  base: '/pg-accounts-management',
  integrations: [
    starlight({
      title: 'DbAccounts',
      description: 'Maintain PostgreSQL roles across many clusters from one desktop app',
      plugins: [starlightThemeRapide()],
      favicon: '/appicon.png',
      logo: {
        src: './src/assets/appicon.png',
        alt: 'DbAccounts',
        replacesTitle: false,
      },
      social: { github: 'https://github.com/michal-bartak/pg-accounts-management' },
      customCss: ['./src/styles/custom.css'],
      // Click a screenshot in the docs body to view it full-size in a lightbox.
      // Runs on first load and after every Starlight client-side navigation.
      head: [
        {
          tag: 'script',
          content: `
            (function () {
              function overlay() {
                var el = document.getElementById('img-lightbox');
                if (el) return el;
                el = document.createElement('div');
                el.id = 'img-lightbox';
                el.className = 'img-lightbox';
                el.innerHTML = '<img alt="">';
                el.addEventListener('click', function () { el.classList.remove('open'); });
                document.addEventListener('keydown', function (e) {
                  if (e.key === 'Escape') el.classList.remove('open');
                });
                document.body.appendChild(el);
                return el;
              }
              function wire() {
                var imgs = document.querySelectorAll('.sl-markdown-content img');
                for (var i = 0; i < imgs.length; i++) {
                  (function (img) {
                    if (img.dataset.lightbox) return;
                    img.dataset.lightbox = '1';
                    img.addEventListener('click', function () {
                      var el = overlay();
                      var big = el.querySelector('img');
                      big.src = img.currentSrc || img.src;
                      big.alt = img.alt || '';
                      el.classList.add('open');
                    });
                  })(imgs[i]);
                }
              }
              document.addEventListener('DOMContentLoaded', wire);
              document.addEventListener('astro:page-load', wire);
            })();
          `,
        },
      ],
      sidebar: [
        { label: 'Overview', link: '/' },
        { label: 'Installation', link: '/installation/' },
        { label: 'Usage', link: '/usage/' },
        { label: 'Call templates', link: '/call-templates/' },
        { label: 'Troubleshooting', link: '/troubleshooting/' },
        { label: 'Building from source', link: '/building/' },
        { label: 'Credits', link: '/credits/' },
      ],
    }),
  ],
});
