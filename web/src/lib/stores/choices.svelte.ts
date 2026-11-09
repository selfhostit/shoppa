import {
  choiceKey,
  OTHER,
  sectionIconKey,
  type Choice,
  type Section,
  type SectionIcon,
} from '../catalogue';
import { upsert, type Item } from '../list';
import { pb } from '../pb';
import { catalogue } from './catalogue.svelte';
import { household } from './household.svelte';
import { run } from './status.svelte';

// How the household sees items: the section and icon they picked for items by
// name, and the icons they picked for sections, on each kind of list.
class Choices {
  /** the section and icon the household picked for items */
  choices = $state<Choice[]>([]);
  /** the icons the household picked for sections */
  sectionIcons = $state<SectionIcon[]>([]);

  /** What the household picked for an item, if they did. */
  choiceOf(item: Item): Choice | undefined {
    const key = choiceKey(item, household.kind);
    return this.choices.find((c) => c.key === key);
  }

  /**
   * An item's icon: the one the household picked, or its catalogue item's, or
   * else its section's. An emoji when that is what was picked before.
   */
  iconOf(item: Item): string {
    const choice = this.choiceOf(item);
    const it = catalogue.catalogueItem(item);
    return (
      choice?.icon ||
      choice?.emoji ||
      it?.icon ||
      it?.emoji ||
      this.sectionIcon(catalogue.kindSections.find((s) => s.key === this.sectionOf(item)))
    );
  }

  /** The section an item is in on the open list: the household's pick, or its own. */
  sectionOf(item: Item): string {
    const fits = (key: string | undefined) =>
      key && catalogue.kindSections.some((s) => s.key === key) ? key : '';
    return (
      fits(this.choiceOf(item)?.section) || fits(catalogue.catalogueItem(item)?.section) || OTHER
    );
  }

  /** A section's icon on the open list: the one the household picked, or its own. */
  sectionIcon(section: Section | undefined): string {
    const key = sectionIconKey(household.kind, section?.key ?? OTHER);
    return (
      this.sectionIcons.find((s) => s.key === key)?.icon ||
      section?.icon ||
      section?.emoji ||
      'package'
    );
  }

  applyChoice(choice: Choice, deleted = false) {
    this.choices = upsert(this.choices, choice, deleted);
  }

  applySectionIcon(icon: SectionIcon, deleted = false) {
    this.sectionIcons = upsert(this.sectionIcons, icon, deleted);
  }

  /**
   * Picks an item's section or icon, for this item and every next one by the
   * same name on this kind of list. An empty one goes back to what the item
   * gets on its own.
   */
  async pick(item: Item, pick: { section?: string; icon?: string }) {
    await run(async () => {
      const was = this.choiceOf(item);
      const next = {
        section: was?.section ?? '',
        icon: was?.icon ?? '',
        emoji: was?.emoji ?? '',
        ...pick,
      };
      // a new icon, or going back to the automatic one, replaces an old emoji
      if ('icon' in pick) next.emoji = '';
      if (was && !next.section && !next.icon && !next.emoji) {
        await pb.collection('item_choices').delete(was.id);
        this.applyChoice(was, true);
      } else if (was) {
        this.applyChoice(await pb.collection<Choice>('item_choices').update(was.id, next));
      } else if (next.section || next.icon || next.emoji) {
        this.applyChoice(
          await pb
            .collection<Choice>('item_choices')
            .create({ key: choiceKey(item, household.kind), ...next }),
        );
      }
    });
  }

  /** Picks the icon a section shows on this kind of list; an empty one goes back to its own. */
  async pickSectionIcon(section: Section | undefined, icon: string) {
    const key = sectionIconKey(household.kind, section?.key ?? OTHER);
    await run(async () => {
      const was = this.sectionIcons.find((s) => s.key === key);
      const own = section?.icon ?? '';
      if (was && (!icon || icon === own)) {
        await pb.collection('section_icons').delete(was.id);
        this.applySectionIcon(was, true);
      } else if (was) {
        this.applySectionIcon(
          await pb.collection<SectionIcon>('section_icons').update(was.id, { icon }),
        );
      } else if (icon && icon !== own) {
        this.applySectionIcon(
          await pb.collection<SectionIcon>('section_icons').create({ key, icon }),
        );
      }
    });
  }
}

export const choices = new Choices();
