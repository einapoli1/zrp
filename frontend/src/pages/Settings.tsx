import { useState, useEffect } from 'react';
import { getSettings, updateSetting, type Setting } from '../lib/api';

export default function Settings() {
  const [settings, setSettings] = useState<Setting[]>([]);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);

  useEffect(() => {
    loadSettings();
  }, []);

  async function loadSettings() {
    try {
      setLoading(true);
      const data = await getSettings();
      setSettings(data);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load settings');
    } finally {
      setLoading(false);
    }
  }

  async function handleToggle(key: string, currentValue: string) {
    setSaving(key);
    setError(null);
    setSuccess(null);
    
    try {
      const newValue = currentValue === 'true' ? 'false' : 'true';
      const updated = await updateSetting(key, newValue);
      
      setSettings(prev => prev.map(s => s.key === key ? updated : s));
      setSuccess('Setting updated successfully');
      setTimeout(() => setSuccess(null), 3000);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to update setting');
    } finally {
      setSaving(null);
    }
  }

  if (loading) {
    return (
      <div className="p-6">
        <div className="animate-pulse">
          <div className="h-8 bg-gray-200 dark:bg-gray-700 rounded w-1/4 mb-4"></div>
          <div className="h-4 bg-gray-200 dark:bg-gray-700 rounded w-1/2"></div>
        </div>
      </div>
    );
  }

  return (
    <div className="p-6 max-w-4xl">
      <h1 className="text-3xl font-bold mb-2 text-gray-900 dark:text-white">Settings</h1>
      <p className="text-gray-600 dark:text-gray-400 mb-6">
        Configure system behavior and workflows
      </p>

      {error && (
        <div className="mb-4 p-4 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-md">
          <p className="text-red-800 dark:text-red-200">{error}</p>
        </div>
      )}

      {success && (
        <div className="mb-4 p-4 bg-green-50 dark:bg-green-900/20 border border-green-200 dark:border-green-800 rounded-md">
          <p className="text-green-800 dark:text-green-200">{success}</p>
        </div>
      )}

      <div className="bg-white dark:bg-gray-800 rounded-lg shadow">
        <div className="p-6 border-b border-gray-200 dark:border-gray-700">
          <h2 className="text-xl font-semibold text-gray-900 dark:text-white">
            Workflow Settings
          </h2>
        </div>

        <div className="divide-y divide-gray-200 dark:divide-gray-700">
          {settings.map((setting) => (
            <div key={setting.key} className="p-6">
              <div className="flex items-center justify-between">
                <div className="flex-1">
                  <h3 className="text-base font-medium text-gray-900 dark:text-white mb-1">
                    {formatSettingName(setting.key)}
                  </h3>
                  <p className="text-sm text-gray-600 dark:text-gray-400">
                    {setting.description}
                  </p>
                  {setting.updated_at && (
                    <p className="text-xs text-gray-500 dark:text-gray-500 mt-2">
                      Last updated: {new Date(setting.updated_at).toLocaleString()}
                    </p>
                  )}
                </div>

                <div className="ml-4">
                  {isBooleanSetting(setting.key) ? (
                    <button
                      type="button"
                      onClick={() => handleToggle(setting.key, setting.value)}
                      disabled={saving === setting.key}
                      className={`
                        relative inline-flex h-6 w-11 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent 
                        transition-colors duration-200 ease-in-out focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2
                        ${setting.value === 'true' ? 'bg-blue-600' : 'bg-gray-200 dark:bg-gray-700'}
                        ${saving === setting.key ? 'opacity-50 cursor-not-allowed' : ''}
                      `}
                      role="switch"
                      aria-checked={setting.value === 'true'}
                    >
                      <span
                        className={`
                          pointer-events-none inline-block h-5 w-5 transform rounded-full bg-white shadow ring-0 
                          transition duration-200 ease-in-out
                          ${setting.value === 'true' ? 'translate-x-5' : 'translate-x-0'}
                        `}
                      />
                    </button>
                  ) : (
                    <span className="text-sm text-gray-600 dark:text-gray-400">
                      {setting.value}
                    </span>
                  )}
                </div>
              </div>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}

function formatSettingName(key: string): string {
  return key
    .split('_')
    .map(word => word.charAt(0).toUpperCase() + word.slice(1))
    .join(' ');
}

function isBooleanSetting(key: string): boolean {
  // Settings that should be rendered as toggles
  return key.startsWith('require_') || key.startsWith('enable_') || key.startsWith('allow_');
}
