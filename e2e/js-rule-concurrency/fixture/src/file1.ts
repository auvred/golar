import { make2, type Box2 } from './file2.ts'

export interface Box1<T> {
	value: T
	map<U>(fn: (value: T) => U): Box1<U>
}

export function make1<T>(value: T): Box1<T> {
	return { value, map: (fn) => make1(fn(value)) }
}

export class Store1<K extends string, V> {
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

export function use1(other: Box2<number>) {
	const store = new Store1<'a' | 'b', Box2<number>>()
	store.set('a', other).set('b', make2(other.value + 1))
	const mapped = make1(other.value)
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
