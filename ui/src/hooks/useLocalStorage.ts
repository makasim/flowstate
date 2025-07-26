import { useState } from "react";

type SerializableValue = string | number | boolean | null | SerializableObject | SerializableArray;
type SerializableObject = { [key: string]: SerializableValue };
type SerializableArray = SerializableValue[];

export function useLocalStorage<T extends SerializableValue>(
  key: string,
  defaultValue: T,
): [T, (newValue: T) => void] {
  const [value, setValue] = useState(() => {
    const savedValue = localStorage.getItem(key);

    if (savedValue === null) return defaultValue;

    try {
      return JSON.parse(savedValue);
    } catch (error) {
      localStorage.removeItem(key);

      return defaultValue;
    }
  });

  function save(newValue: T) {
    setValue(newValue);
    localStorage.setItem(key, JSON.stringify(newValue));
  }

  return [value, save] as const;
}