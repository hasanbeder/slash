import { Option, Select } from "@mui/joy";
import { useTranslation } from "react-i18next";
import BetaBadge from "@/components/BetaBadge";
import { useUserStore } from "@/stores";
import { UserSetting, UserSetting_ColorTheme, UserSetting_Locale } from "@/types/proto/api/v1/user_setting_service";

const PreferenceSection: React.FC = () => {
  const { t } = useTranslation();
  const userStore = useUserStore();
  const userSetting = userStore.getCurrentUserSetting();
  const language = userSetting.locale;
  const colorTheme = userSetting.colorTheme;

  const languageOptions = [
    {
      value: UserSetting_Locale.LOCALE_EN,
      label: "English",
    },
    {
      value: UserSetting_Locale.LOCALE_ZH,
      label: "中文",
    },
  ];

  const colorThemeOptions = [
    {
      value: UserSetting_ColorTheme.COLOR_THEME_SYSTEM,
      label: "System",
    },
    {
      value: UserSetting_ColorTheme.COLOR_THEME_LIGHT,
      label: "Light",
    },
    {
      value: UserSetting_ColorTheme.COLOR_THEME_DARK,
      label: "Dark",
    },
  ];

  const handleSelectLanguage = async (locale: UserSetting_Locale) => {
    await userStore.updateUserSetting(
      {
        ...userSetting,
        locale: locale,
      } as UserSetting,
      ["locale"]
    );
  };

  const handleSelectColorTheme = async (colorTheme: UserSetting_ColorTheme) => {
    await userStore.updateUserSetting(
      {
        ...userSetting,
        colorTheme: colorTheme,
      } as UserSetting,
      ["color_theme"]
    );
  };

  return (
    <div className="w-full flex flex-col sm:flex-row justify-start items-start gap-4 sm:gap-x-16">
      <p className="sm:w-1/4 text-2xl shrink-0 font-semibold text-gray-900 dark:text-gray-500">{t("settings.preference.self")}</p>
      <div className="w-full sm:w-auto grow flex flex-col justify-start items-start gap-4">
        <div className="w-full flex flex-row justify-between items-center">
          <div className="flex flex-row justify-start items-center gap-x-1">
            <span className="dark:text-gray-400">{t("settings.preference.color-theme")}</span>
          </div>
          <Select defaultValue={colorTheme} onChange={(_, value) => handleSelectColorTheme(value as UserSetting_ColorTheme)}>
            {colorThemeOptions.map((option) => {
              return (
                <Option key={option.value} value={option.value}>
                  {option.label}
                </Option>
              );
            })}
          </Select>
        </div>
        <div className="w-full flex flex-row justify-between items-center">
          <div className="flex flex-row justify-start items-center gap-x-1">
            <span className="dark:text-gray-400">{t("common.language")}</span>
            <BetaBadge />
          </div>
          <Select defaultValue={language} onChange={(_, value) => handleSelectLanguage(value as UserSetting_Locale)}>
            {languageOptions.map((option) => {
              return (
                <Option key={option.value} value={option.value}>
                  {option.label}
                </Option>
              );
            })}
          </Select>
        </div>
      </div>
    </div>
  );
};

export default PreferenceSection;
