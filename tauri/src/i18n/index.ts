import i18n from 'i18next';
import { initReactI18next } from 'react-i18next';
import yaml from 'js-yaml';

import enYml from './locales/en.yml?raw';
import zhYml from './locales/zh.yml?raw';
import jpYml from './locales/jp.yml?raw';
import krYml from './locales/kr.yml?raw';
import esYml from './locales/es.yml?raw';
import frYml from './locales/fr.yml?raw';
import deYml from './locales/de.yml?raw';

const en = yaml.load(enYml) as Record<string, unknown>;
const zh = yaml.load(zhYml) as Record<string, unknown>;
const jp = yaml.load(jpYml) as Record<string, unknown>;
const kr = yaml.load(krYml) as Record<string, unknown>;
const es = yaml.load(esYml) as Record<string, unknown>;
const fr = yaml.load(frYml) as Record<string, unknown>;
const de = yaml.load(deYml) as Record<string, unknown>;

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
