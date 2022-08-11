import i18n from "i18next";
import en from './en/en.json';
import zh from './zh/zh.json';
import { initReactI18next } from 'react-i18next';
// import Backend from 'i18next-http-backend';

export const resources = {
    en: {en},
    zh: {zh}
  } as const;

i18n
    // .use(Backend)
    .use(initReactI18next)
    .init({
    // debug: true,
    lng: 'en',
    defaultNS: 'en',
    ns:['zh', 'en'],
    supportedLngs: ['zh', 'en'],
    interpolation:{
        escapeValue: false,
    },
    resources,
    });

i18n.changeLanguage('en', () => {
    const browserLang = window.location.search.substring(1);
    const vars = browserLang.split('&');
    for (let i = 0; i < vars.length; i++) {
        const pair = vars[i].split('=');
        if (pair[0] === 'lang') {
            i18n.setDefaultNamespace(pair[1]);
            i18n.changeLanguage(pair[1]);
        }
    }
})

export default i18n;