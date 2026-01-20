import i18n from 'i18next';
import { initReactI18next } from 'react-i18next';
import en from './locales/en.json';
import zh from './locales/zh.json';
import jp from './locales/jp.json';
import kr from './locales/kr.json';
import es from './locales/es.json';
import fr from './locales/fr.json';
import de from './locales/de.json';

export const resources = {
  en: { translation: en },
  zh: { translation: zh },
  jp: { translation: jp },
  kr: { translation: kr },
  es: { translation: es },
  fr: { translation: fr },
  de: { translation: de },
};

export const supportedLanguages = [
  { code: 'en', name: 'English' },
  { code: 'zh', name: '中文' },
  { code: 'jp', name: '日本語' },
  { code: 'kr', name: '한국어' },
  { code: 'es', name: 'Español' },
  { code: 'fr', name: 'Français' },
  { code: 'de', name: 'Deutsch' },
];

i18n
  .use(initReactI18next)
  .init({
    resources,
    lng: 'en',
    fallbackLng: 'en',
    interpolation: {
      escapeValue: false,
    },
  });

export default i18n;

export function changeLanguage(lang: string) {
  i18n.changeLanguage(lang);
}
