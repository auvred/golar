import { make5, type Box5 } from './file5.ts'

export interface Box4<T> {
	value: T
	map<U>(fn: (value: T) => U): Box4<U>
}

export function make4<T>(value: T): Box4<T> {
	return { value, map: (fn) => make4(fn(value)) }
}

export class Store4<K extends string, V> {
	private readonly items = new Map<K, V>()
	set(key: K, value: V): this {
		this.items.set(key, value)
		return this
	}
	get(key: K): V | undefined {
		return this.items.get(key)
	}
	entries(): Array<[K, V]> {
		return [...this.items.entries()]
	}
}

export function use4(other: Box5<number>) {
	const store = new Store4<'a' | 'b', Box5<number>>()
	store.set('a', other).set('b', make5(other.value + 4))
	const mapped = make4(other.value)
		.map((n) => `${n}`)
		.map((s) => s.length)
	const total = store
		.entries()
		.reduce((sum, [, box]) => sum + box.value, mapped.value)
	const pairs = Object.entries({ x: total, y: mapped.value }).map(([k, v]) => ({
		k,
		v,
	}))
	return pairs.filter(({ v }) => v > 0).map(({ k }) => k.toUpperCase())
}
